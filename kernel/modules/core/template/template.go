package template

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
)

type Code string

type Definition struct {
	Code   Code
	Label  string
	Fields []field.Definition
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
		if _, exists := catalog.runtimes[definition.Code]; exists {
			return nil, fmt.Errorf(
				"duplicate template code %q",
				definition.Code,
			)
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
