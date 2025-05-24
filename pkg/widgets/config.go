package widgets

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"sync"
)

type Config struct {
	Document struct {
		Head template.HTML `yaml:"head"`
	} `yaml:"document"`

	Theme struct {
		Theme   `yaml:",inline"`
		Presets []*Theme `yaml:"presets"`
	} `yaml:"theme"`

	Branding struct {
		HideFooter         bool          `yaml:"hide-footer"`
		CustomFooter       template.HTML `yaml:"custom-footer"`
		LogoText           string        `yaml:"logo-text"`
		LogoURL            string        `yaml:"logo-url"`
		FaviconURL         string        `yaml:"favicon-url"`
		FaviconType        string        `yaml:"-"`
		AppName            string        `yaml:"app-name"`
		AppIconURL         string        `yaml:"app-icon-url"`
		AppBackgroundColor string        `yaml:"app-background-color"`
	} `yaml:"branding"`

	Pages []Page `yaml:"pages"`

	// Computed lookups
	slugToPage map[string]*Page
	widgetByID map[uint64]Widget
}

func NewDefaultConfig() (*Config, error) {
	config := &Config{
		slugToPage: make(map[string]*Page),
		widgetByID: make(map[uint64]Widget),
	}

	config.Theme.Presets = []*Theme{
		{
			Key:                      "default",
			Light:                    true,
			BackgroundColor:          &HSLColor{H: 240, S: 13, L: 95},
			PrimaryColor:             &HSLColor{H: 230, S: 100, L: 30},
			NegativeColor:            &HSLColor{S: 70, L: 50},
			ContrastMultiplier:       1.3,
			TextSaturationMultiplier: 0.5,
		},
	}

	if err := config.Theme.Init(); err != nil {
		return nil, fmt.Errorf("initializing default theme: %v", err)
	}

	return config, nil
}

func (cfg *Config) Init() {
	for p := range cfg.Pages {
		page := &cfg.Pages[p]
		page.PrimaryColumnIndex = -1

		if page.Slug == "" {
			page.Slug = titleToSlug(page.Title)
		}

		cfg.slugToPage[page.Slug] = page

		if page.Width == "default" {
			page.Width = ""
		}

		if page.DesktopNavigationWidth != "" && page.DesktopNavigationWidth != "default" {
			page.DesktopNavigationWidth = page.Width
		}

		for i := range page.HeadWidgets {
			widget := page.HeadWidgets[i]
			cfg.widgetByID[widget.ID()] = widget
		}

		for col := range page.Columns {
			column := &page.Columns[col]

			if page.PrimaryColumnIndex == -1 && column.Size == "full" {
				page.PrimaryColumnIndex = int8(col)
			}

			for w := range column.Widgets {
				widget := column.Widgets[w]
				cfg.widgetByID[widget.ID()] = widget
			}
		}
	}
}

type Page struct {
	Title                  string   `yaml:"name"`
	Slug                   string   `yaml:"slug"`
	Width                  string   `yaml:"width"`
	DesktopNavigationWidth string   `yaml:"desktop-navigation-width"`
	ShowMobileHeader       bool     `yaml:"show-mobile-header"`
	HideDesktopNavigation  bool     `yaml:"hide-desktop-navigation"`
	CenterVertically       bool     `yaml:"center-vertically"`
	HeadWidgets            []Widget `yaml:"head-widgets"`
	Columns                []struct {
		Size    string   `yaml:"size"`
		Widgets []Widget `yaml:"widgets"`
	} `yaml:"columns"`
	PrimaryColumnIndex int8       `yaml:"-"`
	mu                 sync.Mutex `yaml:"-"`
}

var sequentialWhitespacePattern = regexp.MustCompile(`\s+`)

func titleToSlug(s string) string {
	s = strings.ToLower(s)
	s = sequentialWhitespacePattern.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	return s
}

func (cfg *Config) Validate() error {
	if len(cfg.Pages) == 0 {
		return fmt.Errorf("no pages configured")
	}

	for i := range cfg.Pages {
		page := &cfg.Pages[i]

		if page.Title == "" {
			return fmt.Errorf("page %d has no name", i+1)
		}

		if page.Width != "" && (page.Width != "wide" && page.Width != "slim" && page.Width != "default") {
			return fmt.Errorf("page %d: width can only be either wide or slim", i+1)
		}

		if page.DesktopNavigationWidth != "" {
			if page.DesktopNavigationWidth != "wide" && page.DesktopNavigationWidth != "slim" && page.DesktopNavigationWidth != "default" {
				return fmt.Errorf("page %d: desktop-navigation-width can only be either wide or slim", i+1)
			}
		}

		if len(page.Columns) == 0 {
			return fmt.Errorf("page %d has no columns", i+1)
		}

		if page.Width == "slim" {
			if len(page.Columns) > 2 {
				return fmt.Errorf("page %d is slim and cannot have more than 2 columns", i+1)
			}
		} else {
			if len(page.Columns) > 3 {
				return fmt.Errorf("page %d has more than 3 columns", i+1)
			}
		}

		columnSizesCount := make(map[string]int)

		for j := range page.Columns {
			column := &page.Columns[j]

			if column.Size != "small" && column.Size != "full" {
				return fmt.Errorf("column %d of page %d: size can only be either small or full", j+1, i+1)
			}

			columnSizesCount[page.Columns[j].Size]++
		}

		full := columnSizesCount["full"]

		if full > 2 || full == 0 {
			return fmt.Errorf("page %d must have either 1 or 2 full width columns", i+1)
		}
	}

	return nil
}
