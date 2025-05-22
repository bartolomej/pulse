package widgets

import (
	"context"
	"github.com/glanceapp/glance/web"
	"html/template"
	"time"
)

var splitColumnWidgetTemplate = web.MustParseTemplate("split-column.html", "widget-base.html")

type splitColumnWidget struct {
	widgetBase          `yaml:",inline"`
	containerWidgetBase `yaml:",inline"`
	MaxColumns          int `yaml:"max-columns"`
}

func (widget *splitColumnWidget) Initialize() error {
	// TODO(pulse): Refactor error handling
	//widget.withError(nil).withTitle("Split Column").setHideHeader(true)

	if err := widget.containerWidgetBase._initializeWidgets(); err != nil {
		return err
	}

	if widget.MaxColumns < 2 {
		widget.MaxColumns = 2
	}

	return nil
}

func (widget *splitColumnWidget) Update(ctx context.Context) {
	widget.containerWidgetBase._update(ctx)
}

func (widget *splitColumnWidget) SetProviders(providers *WidgetProviders) {
	widget.containerWidgetBase._setProviders(providers)
}

func (widget *splitColumnWidget) RequiresUpdate(now *time.Time) bool {
	return widget.containerWidgetBase._requiresUpdate(now)
}

func (widget *splitColumnWidget) Render() template.HTML {
	return widget.renderTemplate(widget, splitColumnWidgetTemplate)
}
