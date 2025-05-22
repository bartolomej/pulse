package widgets

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type containerWidgetBase struct {
	Widgets Widgets `yaml:"widgets"`
}

func (widget *containerWidgetBase) _initializeWidgets() error {
	for i := range widget.Widgets {
		if err := widget.Widgets[i].Initialize(); err != nil {
			return formatWidgetInitError(err, widget.Widgets[i])
		}
	}

	return nil
}

func (widget *containerWidgetBase) _update(ctx context.Context) {
	var wg sync.WaitGroup
	now := time.Now()

	for w := range widget.Widgets {
		widget := widget.Widgets[w]

		if !widget.RequiresUpdate(&now) {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			widget.Update(ctx)
		}()
	}

	wg.Wait()
}

func (widget *containerWidgetBase) _setProviders(providers *WidgetProviders) {
	for i := range widget.Widgets {
		widget.Widgets[i].SetProviders(providers)
	}
}

func (widget *containerWidgetBase) _requiresUpdate(now *time.Time) bool {
	for i := range widget.Widgets {
		if widget.Widgets[i].RequiresUpdate(now) {
			return true
		}
	}

	return false
}

func formatWidgetInitError(err error, w Widget) error {
	return fmt.Errorf("%s widget: %v", w.Type(), err)
}
