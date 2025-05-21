package widgets

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/glanceapp/glance/pkg/sources"
	"html/template"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

var widgetIDCounter atomic.Uint64

func newWidget(widgetType string) (widget, error) {
	if widgetType == "" {
		return nil, errors.New("widget 'type' property is empty or not specified")
	}

	base := widgetBase{
		ID:  widgetIDCounter.Add(1),
		typ: widgetType,
	}
	var w widget

	switch widgetType {
	case "group":
		w = &groupWidget{widgetBase: base}
	case "split-column":
		w = &splitColumnWidget{widgetBase: base}
	default:
		// widget type is treated as a data source type in this case,
		// which depends on the base widget that renders the generic widget display card
		w = &base
	}

	return w, nil
}

type widgets []widget

func (w *widgets) UnmarshalYAML(node *yaml.Node) error {
	var nodes []yaml.Node

	if err := node.Decode(&nodes); err != nil {
		return err
	}

	for _, node := range nodes {
		meta := struct {
			Type string `yaml:"type"`
		}{}

		if err := node.Decode(&meta); err != nil {
			return err
		}

		widget, err := newWidget(meta.Type)
		if err != nil {
			return fmt.Errorf("line %d: %w", node.Line, err)
		}
		if err = node.Decode(widget); err != nil {
			return err
		}

		if meta.Type != "group" && meta.Type != "split-column" {
			source, err := sources.NewSource(meta.Type)
			if err != nil {
				return fmt.Errorf("line %d: %w", node.Line, err)
			}
			if err = node.Decode(source); err != nil {
				return err
			}

			widget.setSource(source)
		}

		*w = append(*w, widget)
	}

	return nil
}

type widget interface {
	// These need to be exported because they get called in templates
	Render() template.HTML
	Type() string
	GetID() uint64

	initialize() error
	setProviders(*widgetProviders)
	update(context.Context)
	requiresUpdate(now *time.Time) bool
	handleRequest(w http.ResponseWriter, r *http.Request)
	setHideHeader(bool)
	setSource(sources.Source)
}

type feedEntry struct {
	ID          string
	Title       string
	Description string
	URL         string
	ImageURL    string
	PublishedAt time.Time
}

type cacheType int

const (
	cacheTypeInfinite cacheType = iota
	cacheTypeDuration
	cacheTypeOnTheHour
)

type widgetBase struct {
	ID            uint64           `yaml:"-"`
	Providers     *widgetProviders `yaml:"-"`
	typ           string           `yaml:"type"`
	HideHeader    bool             `yaml:"hide-header"`
	CSSClass      string           `yaml:"css-class"`
	Error         error            `yaml:"-"`
	CollapseAfter int              `yaml:"collapse-after"`
	Notice        error            `yaml:"-"`
	// Source TODO(pulse): Temporary store source on a widget. Later it should be stored in a source registry and only passed to the widget for rendering.
	Source         sources.Source `yaml:"-"`
	templateBuffer bytes.Buffer   `yaml:"-"`
}

type widgetProviders struct {
	assetResolver func(string) string
}

func (w *widgetBase) requiresUpdate(now *time.Time) bool {
	if w.Source != nil {
		return w.Source.RequiresUpdate(now)
	}
	return false
}

func (w *widgetBase) update(ctx context.Context) {
	if w.Source != nil {
		w.Source.Update(ctx)
	}
}

func (w *widgetBase) GetID() uint64 {
	return w.ID
}

func (w *widgetBase) setID(id uint64) {
	w.ID = id
}

func (w *widgetBase) setHideHeader(value bool) {
	w.HideHeader = value
}

func (widget *widgetBase) handleRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (w *widgetBase) Type() string {
	return w.typ
}

func (w *widgetBase) setType(t string) {
	w.typ = t
}

func (w *widgetBase) setProviders(providers *widgetProviders) {
	w.Providers = providers
}

func (w *widgetBase) source() sources.Source {
	return w.Source
}

func (w *widgetBase) setSource(s sources.Source) {
	w.Source = s
}

var widgetBaseContentTemplate = mustParseTemplate("widget-base-content.html", "widget-base.html")

func (w *widgetBase) Render() template.HTML {
	return w.renderTemplate(w, widgetBaseContentTemplate)
}

func (w *widgetBase) initialize() error {
	if w.CollapseAfter <= 0 {
		w.CollapseAfter = 3
	}
	return w.Source.Initialize()
}

func (w *widgetBase) renderTemplate(data any, t *template.Template) template.HTML {
	w.templateBuffer.Reset()
	err := t.Execute(&w.templateBuffer, data)
	if err != nil {
		w.Error = err

		slog.Error("Failed to render template", "error", err)

		// need to immediately re-render with the error,
		// otherwise risk breaking the page since the widget
		// will likely be partially rendered with tags not closed.
		w.templateBuffer.Reset()
		err2 := t.Execute(&w.templateBuffer, data)

		if err2 != nil {
			slog.Error("Failed to render error within widget", "error", err2, "initial_error", err)
			w.templateBuffer.Reset()
			// TODO: add some kind of a generic widget error template when the widget
			// failed to render, and we also failed to re-render the widget with the error
		}
	}

	return template.HTML(w.templateBuffer.String())
}
