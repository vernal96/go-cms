package extensions

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/vernal96/go-cms-kernel"
	"github.com/vernal96/go-cms-kernel/eventbus"
)

type exampleResolver struct{ kernel.DatabaseResolver }
type exampleBus struct{ eventbus.Bus }

func TestForeignEntityHookNeedsNoKernelRegistration(t *testing.T) {
	factory, err := kernel.NewProfileRuntimeFactory(exampleResolver{}, kernel.RuntimeServices{EventBus: exampleBus{}, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))})
	if err != nil {
		t.Fatal(err)
	}
	blueprint, err := factory.Compile(context.Background(), kernel.Profile{Code: "example", Modules: []kernel.ProfileModule{{Module: CatalogModule{}}, {Module: CatalogPolicyModule{}}}})
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := blueprint.Build(context.Background(), kernel.NewRuntimeScope("1", "example.test", "en-US", nil))
	if err != nil {
		t.Fatal(err)
	}
	module, ok := runtime.Registry().Module("catalog_example")
	if !ok {
		t.Fatal("catalog missing")
	}
	product, err := module.(*CatalogRuntime).PrepareProduct(context.Background(), Product{})
	if err != nil {
		t.Fatal(err)
	}
	if product.Name != "New product" {
		t.Fatalf("name=%q", product.Name)
	}
}
