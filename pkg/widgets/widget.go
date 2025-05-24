package widgets

import (
	"errors"
	"github.com/glanceapp/glance/pkg/sources/common"
	"html/template"
	"sync/atomic"
)

var widgetIDCounter atomic.Uint64

func NewWidget(widgetType string, feed []common.Activity) (Widget, error) {
	if widgetType == "" {
		return nil, errors.New("widget 'type' property is empty or not specified")
	}

	id := widgetIDCounter.Add(1)

	var w Widget

	switch widgetType {
	case "group":
		w = newWidgetGroup(id, widgetType, feed)
	case "split-column":
		w = newWidgetSplitColumn(id, widgetType, feed)
	default:
		w = newWidgetBase(id, widgetType, feed)
	}

	return w, nil
}

type Widget interface {
	// Render is called within templates.
	Render() template.HTML
	// Type is called within templates.
	Type() string
	// ID is called within templates.
	ID() uint64

	// Initialize is called after the widget is created.
	Initialize() error

	setHideHeader(bool)
}
