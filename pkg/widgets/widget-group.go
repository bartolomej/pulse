package widgets

import (
	"context"
	"errors"
	"github.com/glanceapp/glance/web"
	"html/template"
	"time"
)

var groupWidgetTemplate = web.MustParseTemplate("group.html", "widget-base.html")

type groupWidget struct {
	widgetBase          `yaml:",inline"`
	containerWidgetBase `yaml:",inline"`
}

func (widget *groupWidget) Initialize() error {
	// TODO(pulse): Refactor error handling
	//widget.withError(nil)
	widget.HideHeader = true

	for i := range widget.Widgets {
		widget.Widgets[i].setHideHeader(true)

		if widget.Widgets[i].Type() == "group" {
			return errors.New("nested groups are not supported")
		} else if widget.Widgets[i].Type() == "split-column" {
			return errors.New("split columns inside of groups are not supported")
		}
	}

	if err := widget.containerWidgetBase._initializeWidgets(); err != nil {
		return err
	}

	return nil
}

func (widget *groupWidget) Update(ctx context.Context) {
	widget.containerWidgetBase._update(ctx)
}

func (widget *groupWidget) SetProviders(providers *WidgetProviders) {
	widget.containerWidgetBase._setProviders(providers)
}

func (widget *groupWidget) RequiresUpdate(now *time.Time) bool {
	return widget.containerWidgetBase._requiresUpdate(now)
}

func (widget *groupWidget) Render() template.HTML {
	return widget.renderTemplate(widget, groupWidgetTemplate)
}
