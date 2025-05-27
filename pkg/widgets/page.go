package widgets

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type Page struct {
	Title                  string   `json:"name"`
	Slug                   string   `json:"slug"`
	Width                  string   `json:"width"`
	DesktopNavigationWidth string   `json:"desktop_navigation_width"`
	ShowMobileHeader       bool     `json:"show_mobile_header"`
	HideDesktopNavigation  bool     `json:"hide_desktop_navigation"`
	CenterVertically       bool     `json:"center_vertically"`
	HeadWidgets            []Widget `json:"head_widgets"`
	Columns                []struct {
		Size    string   `json:"size"`
		Widgets []Widget `json:"widgets"`
	} `json:"columns"`
	PrimaryColumnIndex int8 `json:"-"`
}

func (page *Page) ID() string {
	return page.Slug
}

func NewPageFromJSON(in []byte) (*Page, error) {
	page := &Page{}

	// TODO(pulse): Fix widget json deserialization
	if err := json.Unmarshal(in, page); err != nil {
		return nil, fmt.Errorf("unmarshal page: %v", err)
	}

	page.Init()

	if err := page.Validate(); err != nil {
		return nil, fmt.Errorf("validate page: %v", err)
	}

	return page, nil
}

func (page *Page) Init() {
	page.PrimaryColumnIndex = -1

	if page.Slug == "" {
		page.Slug = titleToSlug(page.Title)
	}

	if page.Width == "default" {
		page.Width = ""
	}

	if page.DesktopNavigationWidth != "" && page.DesktopNavigationWidth != "default" {
		page.DesktopNavigationWidth = page.Width
	}

	for col := range page.Columns {
		column := &page.Columns[col]

		if page.PrimaryColumnIndex == -1 && column.Size == "full" {
			page.PrimaryColumnIndex = int8(col)
		}
	}
}

func (page *Page) Validate() error {
	if page.Title == "" {
		return fmt.Errorf("page %s has no name", page.ID())
	}

	if page.Width != "" && (page.Width != "wide" && page.Width != "slim" && page.Width != "default") {
		return fmt.Errorf("page %s: width can only be either wide or slim", page.ID())
	}

	if page.DesktopNavigationWidth != "" {
		if page.DesktopNavigationWidth != "wide" && page.DesktopNavigationWidth != "slim" && page.DesktopNavigationWidth != "default" {
			return fmt.Errorf("page %s: desktop-navigation-width can only be either wide or slim", page.ID())
		}
	}

	if len(page.Columns) == 0 {
		return fmt.Errorf("page %s has no columns", page.ID())
	}

	if page.Width == "slim" {
		if len(page.Columns) > 2 {
			return fmt.Errorf("page %s is slim and cannot have more than 2 columns", page.ID())
		}
	} else {
		if len(page.Columns) > 3 {
			return fmt.Errorf("page %s has more than 3 columns", page.ID())
		}
	}

	columnSizesCount := make(map[string]int)

	for j := range page.Columns {
		column := &page.Columns[j]

		if column.Size != "small" && column.Size != "full" {
			return fmt.Errorf("column %d of page %s: size can only be either small or full", j+1, page.ID())
		}

		columnSizesCount[page.Columns[j].Size]++
	}

	full := columnSizesCount["full"]

	if full > 2 || full == 0 {
		return fmt.Errorf("page %s must have either 1 or 2 full width columns", page.ID())
	}

	return nil
}

var sequentialWhitespacePattern = regexp.MustCompile(`\s+`)

func titleToSlug(s string) string {
	s = strings.ToLower(s)
	s = sequentialWhitespacePattern.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	return s
}
