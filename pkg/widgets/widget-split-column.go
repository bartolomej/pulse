package widgets

import (
	"github.com/glanceapp/glance/pkg/sources/common"
	"github.com/glanceapp/glance/web"
	"html/template"
)

var splitColumnWidgetTemplate = web.MustParseTemplate("split-column.html", "widget-base.html")

type splitColumnWidget struct {
	widgetBase          `yaml:",inline"`
	containerWidgetBase `yaml:",inline"`
	MaxColumns          int `yaml:"max-columns"`
}

func newWidgetSplitColumn(id uint64, typ string, feed []common.Activity) *splitColumnWidget {
	return &splitColumnWidget{
		widgetBase:          newWidgetBase(id, typ, feed),
		containerWidgetBase: containerWidgetBase{},
		MaxColumns:          2,
	}
}

func (w *splitColumnWidget) Initialize() error {
	// TODO(pulse): Refactor error handling
	//widget.withError(nil).withTitle("Split Column").setHideHeader(true)

	if err := w.containerWidgetBase._initializeWidgets(); err != nil {
		return err
	}

	if w.MaxColumns < 2 {
		w.MaxColumns = 2
	}

	return nil
}

func (w *splitColumnWidget) Render() template.HTML {
	return w.renderTemplate(w, splitColumnWidgetTemplate)
}
