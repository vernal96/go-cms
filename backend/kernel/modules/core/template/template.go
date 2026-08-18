package template

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
)

type Code string

type ItemKind string

const (
	ItemWidget       ItemKind = "widget"
	ItemResourceSlot ItemKind = "resource_widgets"
)

type LayoutItem struct {
	Kind         ItemKind
	Key          string
	Widget       widget.Code
	Presentation widget.Presentation
	Params       map[string]any
}

type Layout struct {
	Body    []LayoutItem
	Sidebar []LayoutItem
}

type Definition struct {
	Code   Code
	Label  string
	Icon   string
	Fields []field.Definition
	Layout Layout
}

type Runtime struct {
	definition Definition
	schema     *field.Schema
}

func (r *Runtime) Definition() Definition {
	if r == nil {
		return Definition{}
	}

	return CloneDefinition(r.definition)
}

func (r *Runtime) FieldSchema() *field.Schema {
	if r == nil {
		return nil
	}

	return r.schema
}

type Catalog struct {
	order    []Code
	runtimes map[Code]*Runtime
}

func Compile(
	definitions []Definition,
	resolver field.TypeResolver,
) (*Catalog, error) {
	if resolver == nil {
		return nil, errors.New("template field type resolver is nil")
	}

	definitions = CloneDefinitions(definitions)
	catalog := &Catalog{
		order:    make([]Code, 0, len(definitions)),
		runtimes: make(map[Code]*Runtime, len(definitions)),
	}

	for index, definition := range definitions {
		if definition.Code == "" ||
			strings.TrimSpace(string(definition.Code)) !=
				string(definition.Code) {
			return nil, fmt.Errorf(
				"template at index %d has invalid code %q",
				index,
				definition.Code,
			)
		}
		if definition.Label == "" ||
			strings.TrimSpace(definition.Label) != definition.Label {
			return nil, fmt.Errorf(
				"template %q has invalid label %q",
				definition.Code,
				definition.Label,
			)
		}
		definition.Icon = strings.TrimSpace(definition.Icon)
		if strings.ContainsAny(definition.Icon, " /\\") {
			return nil, fmt.Errorf(
				"template %q has invalid icon %q",
				definition.Code,
				definition.Icon,
			)
		}
		if _, exists := catalog.runtimes[definition.Code]; exists {
			return nil, fmt.Errorf(
				"duplicate template code %q",
				definition.Code,
			)
		}
		if err := validateLayout(definition.Code, definition.Layout); err != nil {
			return nil, err
		}

		schema, err := field.Compile(definition.Fields, resolver)
		if err != nil {
			return nil, fmt.Errorf(
				"compile template %q fields: %w",
				definition.Code,
				err,
			)
		}

		catalog.order = append(catalog.order, definition.Code)
		catalog.runtimes[definition.Code] = &Runtime{
			definition: definition,
			schema:     schema,
		}
	}

	return catalog, nil
}

func validateLayout(code Code, layout Layout) error {
	keys := make(map[string]struct{})
	for _, area := range []struct {
		code  widget.AreaCode
		items []LayoutItem
	}{
		{code: widget.AreaBody, items: layout.Body},
		{code: widget.AreaSidebar, items: layout.Sidebar},
	} {
		slots := 0
		for index, item := range area.items {
			switch item.Kind {
			case ItemResourceSlot:
				slots++
				if item.Key != "" || item.Widget != "" || len(item.Params) != 0 ||
					item.Presentation != (widget.Presentation{}) {
					return fmt.Errorf("template %q %s resource slot at index %d has widget data", code, area.code, index)
				}
			case ItemWidget:
				if item.Key == "" || strings.TrimSpace(item.Key) != item.Key {
					return fmt.Errorf("template %q %s widget at index %d has invalid key %q", code, area.code, index, item.Key)
				}
				if _, exists := keys[item.Key]; exists {
					return fmt.Errorf("template %q has duplicate widget key %q", code, item.Key)
				}
				keys[item.Key] = struct{}{}
				if item.Widget == "" || strings.TrimSpace(string(item.Widget)) != string(item.Widget) {
					return fmt.Errorf("template %q %s widget %q has invalid code", code, area.code, item.Key)
				}
				if err := item.Presentation.Validate(); err != nil {
					return fmt.Errorf("template %q %s widget %q: %w", code, area.code, item.Key, err)
				}
			default:
				return fmt.Errorf("template %q %s item at index %d has invalid kind %q", code, area.code, index, item.Kind)
			}
		}
		if slots > 1 {
			return fmt.Errorf("template %q %s contains duplicate resource widget slots", code, area.code)
		}
	}
	return nil
}

func (r *Runtime) SupportsResourceWidgets() bool {
	return len(r.ResourceAreas()) > 0
}

func (r *Runtime) ResourceAreas() []widget.AreaCode {
	if r == nil {
		return nil
	}
	result := make([]widget.AreaCode, 0, 2)
	if hasResourceSlot(r.definition.Layout.Body) {
		result = append(result, widget.AreaBody)
	}
	if hasResourceSlot(r.definition.Layout.Sidebar) {
		result = append(result, widget.AreaSidebar)
	}
	return result
}

func (r *Runtime) AllowsResourceArea(area widget.AreaCode) bool {
	for _, current := range r.ResourceAreas() {
		if current == area {
			return true
		}
	}
	return false
}

func hasResourceSlot(items []LayoutItem) bool {
	for _, item := range items {
		if item.Kind == ItemResourceSlot {
			return true
		}
	}
	return false
}

// ValidateWidgets performs final reference validation after the site-scoped
// widget catalog has been assembled from enabled module runtimes.
func (c *Catalog) ValidateWidgets(catalog interface {
	Widget(widget.Code) (*widget.Runtime, bool)
}) error {
	if c == nil || catalog == nil {
		return errors.New("template widget catalog is unavailable")
	}
	for _, code := range c.order {
		definition := c.runtimes[code].definition
		for _, area := range [][]LayoutItem{definition.Layout.Body, definition.Layout.Sidebar} {
			for _, item := range area {
				if item.Kind != ItemWidget {
					continue
				}
				runtime, exists := catalog.Widget(item.Widget)
				if !exists {
					return fmt.Errorf("template %q references unavailable widget %q", code, item.Widget)
				}
				if err := runtime.ValidatePresentation(item.Presentation); err != nil {
					return fmt.Errorf("template %q widget %q: %w", code, item.Key, err)
				}
				if _, err := runtime.NormalizeParams(item.Params); err != nil {
					return fmt.Errorf("template %q widget %q: %w", code, item.Key, err)
				}
			}
		}
	}
	return nil
}

func Compose(runtime *Runtime, bindings []widget.Binding) (widget.Placements, error) {
	if runtime == nil {
		return widget.Placements{}, errors.New("template runtime is nil")
	}
	byArea := map[widget.AreaCode][]widget.Binding{
		widget.AreaBody: {}, widget.AreaSidebar: {},
	}
	for _, binding := range bindings {
		if !widget.ValidArea(binding.Area) {
			return widget.Placements{}, fmt.Errorf("resource widget %d has invalid area %q", binding.ID, binding.Area)
		}
		if !runtime.AllowsResourceArea(binding.Area) {
			return widget.Placements{}, fmt.Errorf("template %q does not allow resource widgets in %q", runtime.definition.Code, binding.Area)
		}
		byArea[binding.Area] = append(byArea[binding.Area], widget.CloneBinding(binding))
	}
	for area := range byArea {
		sort.SliceStable(byArea[area], func(left, right int) bool {
			return byArea[area][left].Position < byArea[area][right].Position
		})
		for index, binding := range byArea[area] {
			if binding.Position != index {
				return widget.Placements{}, fmt.Errorf("resource widget %d in %q has position %d instead of %d", binding.ID, area, binding.Position, index)
			}
		}
	}
	body, err := composeArea(widget.AreaBody, runtime.definition.Layout.Body, byArea[widget.AreaBody])
	if err != nil {
		return widget.Placements{}, err
	}
	sidebar, err := composeArea(widget.AreaSidebar, runtime.definition.Layout.Sidebar, byArea[widget.AreaSidebar])
	if err != nil {
		return widget.Placements{}, err
	}
	return widget.Placements{Body: body, Sidebar: sidebar}, nil
}

func composeArea(area widget.AreaCode, items []LayoutItem, bindings []widget.Binding) ([]widget.Placement, error) {
	result := make([]widget.Placement, 0, len(items)+len(bindings))
	for _, item := range items {
		switch item.Kind {
		case ItemWidget:
			result = append(result, widget.Placement{
				Key: item.Key, Code: item.Widget, Area: area,
				Presentation: item.Presentation, Params: cloneMap(item.Params),
			})
		case ItemResourceSlot:
			for _, binding := range bindings {
				result = append(result, widget.Placement{
					Key: fmt.Sprintf("resource-widget-%d", binding.ID), BindingID: binding.ID,
					Code: binding.Code, Area: area, Position: binding.Position,
					Presentation: binding.Presentation,
					Params:       cloneMap(binding.Params),
				})
			}
		default:
			return nil, fmt.Errorf("template area %q contains invalid item kind %q", area, item.Kind)
		}
	}
	return result, nil
}

func (c *Catalog) Template(code Code) (*Runtime, bool) {
	if c == nil {
		return nil, false
	}

	runtime, exists := c.runtimes[code]
	return runtime, exists
}

func (c *Catalog) Definitions() []Definition {
	if c == nil {
		return nil
	}

	result := make([]Definition, 0, len(c.order))
	for _, code := range c.order {
		result = append(
			result,
			CloneDefinition(c.runtimes[code].definition),
		)
	}

	return result
}

func CloneDefinition(definition Definition) Definition {
	definition.Fields = field.CloneDefinitions(definition.Fields)
	definition.Layout = Layout{
		Body:    cloneLayoutItems(definition.Layout.Body),
		Sidebar: cloneLayoutItems(definition.Layout.Sidebar),
	}
	return definition
}

func CloneDefinitions(source []Definition) []Definition {
	if source == nil {
		return nil
	}

	result := make([]Definition, len(source))
	for index, definition := range source {
		result[index] = CloneDefinition(definition)
	}
	return result
}

func cloneLayoutItems(source []LayoutItem) []LayoutItem {
	if source == nil {
		return nil
	}
	result := make([]LayoutItem, len(source))
	for index, item := range source {
		result[index] = item
		result[index].Params = cloneMap(item.Params)
	}
	return result
}

func cloneMap(source map[string]any) map[string]any {
	if source == nil {
		return nil
	}
	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
