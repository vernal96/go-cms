package widget

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
)

type Code string

var (
	ErrInvalidParams  = errors.New("invalid widget params")
	ErrInstanceFailed = errors.New("widget instance failed")
)

type Definition struct {
	Code        Code
	Label       string
	Description string
	Fields      []field.Definition
}

// Widget is a module-owned widget definition. Definition.Code is local to
// the owning module; Catalog qualifies it before exposing it to a profile.
type Widget interface {
	Definition() Definition
	New(map[string]any) (Instance, error)
}

// ResourceSnapshot is the resource data available to a widget during render.
// It deliberately lives in this package so widgets do not depend on resource
// persistence types.
type ResourceSnapshot struct {
	ID      int64
	Content string
}

type RenderInput struct {
	Resource ResourceSnapshot
}

type Instance interface {
	Render(context.Context, RenderInput) (map[string]any, error)
}

// Provider is implemented optionally by a module runtime.
type Provider interface {
	Widgets() []Widget
}

type Source struct {
	Module  string
	Widgets []Widget
}

type Runtime struct {
	definition Definition
	schema     *field.Schema
	widget     Widget
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

func (r *Runtime) New(values map[string]any) (Instance, error) {
	if r == nil {
		return nil, errors.New("widget runtime is nil")
	}

	normalized, err := r.schema.Validate(values)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidParams, err)
	}

	instance, err := r.widget.New(normalized)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInstanceFailed, err)
	}
	if isNil(instance) {
		return nil, fmt.Errorf(
			"%w: widget returned nil instance",
			ErrInstanceFailed,
		)
	}

	return instance, nil
}

type Catalog struct {
	order    []Code
	runtimes map[Code]*Runtime
}

func Compile(
	sources []Source,
	resolver field.TypeResolver,
) (*Catalog, error) {
	if resolver == nil {
		return nil, errors.New("widget field type resolver is nil")
	}

	catalog := &Catalog{
		runtimes: make(map[Code]*Runtime),
	}

	for sourceIndex, source := range sources {
		if source.Module == "" ||
			strings.TrimSpace(source.Module) != source.Module {
			return nil, fmt.Errorf(
				"widget source at index %d has invalid module code %q",
				sourceIndex,
				source.Module,
			)
		}

		localCodes := make(map[Code]struct{}, len(source.Widgets))
		for widgetIndex, current := range source.Widgets {
			if isNil(current) {
				return nil, fmt.Errorf(
					"module %q widget at index %d is nil",
					source.Module,
					widgetIndex,
				)
			}

			declaration := CloneDefinition(current.Definition())
			localCode := declaration.Code
			if localCode == "" ||
				strings.TrimSpace(string(localCode)) != string(localCode) {
				return nil, fmt.Errorf(
					"module %q widget at index %d has invalid code %q",
					source.Module,
					widgetIndex,
					localCode,
				)
			}
			if declaration.Label == "" ||
				strings.TrimSpace(declaration.Label) != declaration.Label {
				return nil, fmt.Errorf(
					"module %q widget %q has invalid label %q",
					source.Module,
					localCode,
					declaration.Label,
				)
			}
			if declaration.Description == "" ||
				strings.TrimSpace(declaration.Description) !=
					declaration.Description {
				return nil, fmt.Errorf(
					"module %q widget %q has invalid description %q",
					source.Module,
					localCode,
					declaration.Description,
				)
			}
			if _, exists := localCodes[localCode]; exists {
				return nil, fmt.Errorf(
					"module %q contains duplicate widget code %q",
					source.Module,
					localCode,
				)
			}
			localCodes[localCode] = struct{}{}

			globalCode := Code(source.Module + "_" + string(localCode))
			if _, exists := catalog.runtimes[globalCode]; exists {
				return nil, fmt.Errorf(
					"duplicate global widget code %q",
					globalCode,
				)
			}

			schema, err := field.Compile(declaration.Fields, resolver)
			if err != nil {
				return nil, fmt.Errorf(
					"compile widget %q fields: %w",
					globalCode,
					err,
				)
			}

			declaration.Code = globalCode
			catalog.order = append(catalog.order, globalCode)
			catalog.runtimes[globalCode] = &Runtime{
				definition: declaration,
				schema:     schema,
				widget:     current,
			}
		}
	}

	return catalog, nil
}

func (c *Catalog) Widget(code Code) (*Runtime, bool) {
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

func isNil(value any) bool {
	if value == nil {
		return true
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
