// Package extensions is a compilable example of a project module. Register
// Module in a profile; no kernel or HTTP server edits are needed.
package extensions

import (
	"context"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/modules/forms"
)

const Code kernel.ModuleCode = "example"

var Greeting = widget.NewRef("greeting")

type Module struct{}

func (Module) Code() kernel.ModuleCode { return Code }
func (Module) Build(context.Context, kernel.ModuleContext) (kernel.ModuleRuntime, error) {
	return Runtime{}, nil
}

type Runtime struct{}

func (Runtime) ModuleCode() kernel.ModuleCode { return Code }
func (Runtime) Widgets() []widget.Widget {
	return []widget.Widget{widget.Functional{
		Description: widget.Definition{Reference: Greeting, Label: "Приветствие", Description: "Приветствует посетителя по имени", Fields: []field.Definition{{Key: "name", Label: "Имя", Type: field.TypeString}}},
		Render: func(_ context.Context, _ widget.RenderInput, params map[string]any) (map[string]any, error) {
			name, _ := params["name"].(string)
			return map[string]any{"greeting": "Здравствуйте, " + name}, nil
		},
	}}
}
func Page() template.Definition {
	return template.Definition{Code: "greeting", Label: "Приветствие", Layout: template.Layout{Body: []template.Item{template.Widget{Widget: Greeting}, template.ResourceWidgets{}}}}
}

// RegisterElements can be called from a module Build after declaring a Forms
// dependency and resolving forms.ElementRegistrar with ModuleDependencyFrom.
func RegisterElements(registrar forms.ElementRegistrar) error {
	return registrar.RegisterElementType(forms.ElementDefinition{Description: forms.ElementTypeMetadata{
		Code: "example.notice", Label: "Уведомление", Fields: []field.ConfigField{
			{Key: "text", Label: "Текст", Type: field.TypeTextarea, Required: true},
			{Key: "highlighted", Label: "Выделить", Type: field.TypeCheckbox},
		},
	}})
}
