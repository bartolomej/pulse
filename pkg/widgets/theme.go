package widgets

import (
	"bytes"
	"fmt"
	"github.com/glanceapp/glance/web"
	"html/template"
	"regexp"
)

var (
	themeStyleTemplate         = web.MustParseTemplate("theme-style.gotmpl")
	themePresetPreviewTemplate = web.MustParseTemplate("theme-preset-preview.html")
)

type Theme struct {
	BackgroundColor          *HSLColor `yaml:"background-color"`
	PrimaryColor             *HSLColor `yaml:"primary-color"`
	PositiveColor            *HSLColor `yaml:"positive-color"`
	NegativeColor            *HSLColor `yaml:"negative-color"`
	Light                    bool      `yaml:"light"`
	ContrastMultiplier       float32   `yaml:"contrast-multiplier"`
	TextSaturationMultiplier float32   `yaml:"text-saturation-multiplier"`

	Key                  string        `yaml:"-"`
	CSS                  template.CSS  `yaml:"-"`
	PreviewHTML          template.HTML `yaml:"-"`
	BackgroundColorAsHex string        `yaml:"-"`
}

var whitespaceAtBeginningOfLinePattern = regexp.MustCompile(`(?m)^\s+`)

func (t *Theme) Init() error {
	css, err := executeTemplateToString(themeStyleTemplate, t)
	if err != nil {
		return fmt.Errorf("compiling theme style: %v", err)
	}
	t.CSS = template.CSS(whitespaceAtBeginningOfLinePattern.ReplaceAllString(css, ""))

	previewHTML, err := executeTemplateToString(themePresetPreviewTemplate, t)
	if err != nil {
		return fmt.Errorf("compiling theme preview: %v", err)
	}
	t.PreviewHTML = template.HTML(previewHTML)

	if t.BackgroundColor != nil {
		t.BackgroundColorAsHex = t.BackgroundColor.ToHex()
	} else {
		t.BackgroundColorAsHex = "#151519"
	}

	return nil
}

func (t1 *Theme) SameAs(t2 *Theme) bool {
	if t1 == nil && t2 == nil {
		return true
	}
	if t1 == nil || t2 == nil {
		return false
	}
	if t1.Light != t2.Light {
		return false
	}
	if t1.ContrastMultiplier != t2.ContrastMultiplier {
		return false
	}
	if t1.TextSaturationMultiplier != t2.TextSaturationMultiplier {
		return false
	}
	if t1.BackgroundColor != t2.BackgroundColor {
		return false
	}
	if t1.PrimaryColor != t2.PrimaryColor {
		return false
	}
	if t1.PositiveColor != t2.PositiveColor {
		return false
	}
	if t1.NegativeColor != t2.NegativeColor {
		return false
	}
	return true
}

func executeTemplateToString(t *template.Template, data any) (string, error) {
	var b bytes.Buffer
	err := t.Execute(&b, data)
	if err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return b.String(), nil
}
