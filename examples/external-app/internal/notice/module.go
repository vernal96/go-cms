package notice

import (
	"context"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/adminui"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/forms"
)

type Module struct{}

func (Module) Code() kernel.ModuleCode           { return "example" }
func (Module) Dependencies() []kernel.ModuleCode { return []kernel.ModuleCode{forms.ModuleCode} }
func (Module) Registry() kernel.ModuleRegistry {
	return kernel.ModuleRegistry{FieldTypes: []field.Type{field.DescribedType{Type: textType{}, Presentation: field.Metadata{Label: "Example text", Editor: "example.text"}}}}
}
func (Module) Build(_ context.Context, ctx kernel.ModuleContext) (kernel.ModuleRuntime, error) {
	registrar, err := kernel.ModuleDependencyFrom[interface {
		kernel.ModuleRuntime
		forms.ElementRegistrar
	}](ctx, forms.ModuleCode)
	if err != nil {
		return nil, err
	}
	err = registrar.RegisterElementType(forms.ElementDefinition{FieldTypes: ctx.Registry(), Description: forms.ElementTypeMetadata{
		Code: "example.notice", Label: "Example notice", EditorCode: "example.notice",
		Fields: []field.ConfigField{{Key: "text", Label: "Text", Type: "example.text", Editor: "example.text", Required: true}},
	}})
	if err != nil {
		return nil, err
	}
	return Runtime{SiteID: ctx.Scope().SiteID()}, nil
}

type Runtime struct{ SiteID string }

func (Runtime) ModuleCode() kernel.ModuleCode { return "example" }
func (Runtime) AdminNavigation() []adminui.NavigationItem {
	return []adminui.NavigationItem{{Code: "example.notice", Label: "Example notice", Route: "example.notice", Scope: adminui.NavigationSite, Order: 80}}
}

type textType struct{}

func (textType) Code() field.TypeCode { return "example.text" }
func (textType) Compile(ctx field.CompileContext, options any) (field.ValueType, error) {
	base, _ := field.StandardTypes().FieldType(field.TypeString)
	return base.Compile(ctx, options)
}
