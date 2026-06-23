package core

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type runtimeWidgetRegistry struct {
	moduleCode string
	widgets    map[WidgetCode]Widget
}

func (r *runtimeWidgetRegistry) Register(widget Widget) error {
	if widget == nil {
		return errors.New("widget is nil")
	}

	code, err := r.fullWidgetCode(widget.Code())
	if err != nil {
		return err
	}

	if _, exists := r.widgets[code]; exists {
		return fmt.Errorf("widget %q is already registered", code)
	}

	r.widgets[code] = widget

	return nil
}

func (r *runtimeWidgetRegistry) Get(code WidgetCode) (Widget, bool) {
	widget, exists := r.widgets[code]

	return widget, exists
}

func (r *runtimeWidgetRegistry) All() []Widget {
	widgets := make([]Widget, 0, len(r.widgets))

	for _, widget := range r.widgets {
		widgets = append(widgets, widget)
	}

	slices.SortFunc(widgets, func(a, b Widget) int {
		return strings.Compare(string(a.Code()), string(b.Code()))
	})

	return widgets
}

func (r *runtimeWidgetRegistry) fullWidgetCode(code WidgetCode) (WidgetCode, error) {
	if code == "" {
		return "", errors.New("widget code is empty")
	}

	if strings.Contains(string(code), ".") {
		return "", fmt.Errorf("widget code %q must be local", code)
	}

	if r.moduleCode == "" {
		return "", errors.New("module code is empty")
	}

	return WidgetCode(r.moduleCode + "." + string(code)), nil
}

var _ WidgetRegistry = (*runtimeWidgetRegistry)(nil)
