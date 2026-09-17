package extensions

import (
	"context"
	"github.com/vernal96/go-cms-kernel/modules/core/field"
	"github.com/vernal96/go-cms-kernel/modules/core/template"
	"github.com/vernal96/go-cms-kernel/modules/core/widget"
	"testing"
)

func TestExampleDeclarationsCompileAndRender(t *testing.T) {
	catalog, err := widget.Compile([]widget.Source{{Module: widget.ModuleDescriptor{Code: string(Code), Label: "Example"}, Widgets: Runtime{}.Widgets()}}, nil, field.StandardTypes())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := template.Compile([]template.Definition{Page()}, field.StandardTypes()); err != nil {
		t.Fatal(err)
	}
	definition := catalog.Definitions()[0]
	runtime, ok := catalog.Widget(definition.Code)
	if !ok {
		t.Fatal("widget unavailable")
	}
	instance, err := runtime.New(map[string]any{"name": "Анна"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := instance.Render(context.Background(), widget.RenderInput{})
	if err != nil {
		t.Fatal(err)
	}
	if result["greeting"] != "Здравствуйте, Анна" {
		t.Fatalf("result=%v", result)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := instance.Render(ctx, widget.RenderInput{}); err == nil {
		t.Fatal("cancellation was ignored")
	}
}
