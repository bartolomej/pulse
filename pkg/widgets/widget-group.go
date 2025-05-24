package widgets

import (
	"errors"
	"github.com/glanceapp/glance/pkg/sources/common"
	"github.com/glanceapp/glance/web"
	"html/template"
)

var groupWidgetTemplate = web.MustParseTemplate("group.html", "widget-base.html")

type groupWidget struct {
	widgetBase          `yaml:",inline"`
	containerWidgetBase `yaml:",inline"`
}

func newWidgetGroup(id uint64, typ string, feed []common.Activity) *groupWidget {
	return &groupWidget{
		widgetBase:          newWidgetBase(id, typ, feed),
		containerWidgetBase: containerWidgetBase{},
	}
}

func (w *groupWidget) Initialize() error {
	// TODO(pulse): Refactor error handling
	//widget.withError(nil)
	w.HideHeader = true

	for i := range w.Widgets {
		w.Widgets[i].setHideHeader(true)

		if w.Widgets[i].Type() == "group" {
			return errors.New("nested groups are not supported")
		} else if w.Widgets[i].Type() == "split-column" {
			return errors.New("split columns inside of groups are not supported")
		}
	}

	if err := w.containerWidgetBase._initializeWidgets(); err != nil {
		return err
	}

	return nil
}

func (w *groupWidget) Render() template.HTML {
	return w.renderTemplate(w, groupWidgetTemplate)
}
