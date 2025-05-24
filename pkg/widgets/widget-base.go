package widgets

import (
	"bytes"
	"github.com/glanceapp/glance/pkg/sources/common"
	"github.com/glanceapp/glance/web"
	"html/template"
)

type widgetBase struct {
	id             uint64
	Typ            string `json:"typ"`
	HideHeader     bool   `json:"hide_header"`
	CSSClass       string `json:"css_class"`
	CollapseAfter  int    `json:"collapse_after"`
	Error          error
	Notice         error
	Feed           []common.Activity
	templateBuffer bytes.Buffer
}

func newWidgetBase(id uint64, typ string, feed []common.Activity) *widgetBase {
	return &widgetBase{
		id:             id,
		Typ:            typ,
		HideHeader:     false,
		CSSClass:       "",
		Error:          nil,
		CollapseAfter:  3,
		Notice:         nil,
		Feed:           feed,
		templateBuffer: bytes.Buffer{},
	}
}

func (w *widgetBase) Type() string {
	return w.Typ
}

func (w *widgetBase) ID() uint64 {
	return w.id
}

func (w *widgetBase) setHideHeader(value bool) {
	w.HideHeader = value
}

var widgetBaseContentTemplate = web.MustParseTemplate("widget-base-content.html", "widget-base.html")

func (w *widgetBase) Render() template.HTML {
	return w.renderTemplate(w, widgetBaseContentTemplate)
}

func (w *widgetBase) Initialize() error {
	if w.CollapseAfter <= 0 {
		w.CollapseAfter = 3
	}
	return nil
}

func (w *widgetBase) renderTemplate(data any, t *template.Template) template.HTML {
	w.templateBuffer.Reset()
	err := t.Execute(&w.templateBuffer, data)
	if err != nil {
		w.Error = err

		// need to immediately re-render with the error,
		// otherwise risk breaking the page since the widget
		// will likely be partially rendered with tags not closed.
		w.templateBuffer.Reset()
		err2 := t.Execute(&w.templateBuffer, data)

		if err2 != nil {
			w.templateBuffer.Reset()
			// TODO: add some kind of a generic widget error template when the widget
			// failed to render, and we also failed to re-render the widget with the error
		}
	}

	return template.HTML(w.templateBuffer.String())
}
