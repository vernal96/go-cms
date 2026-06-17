package testmodule

import (
	"context"
	"fmt"

	"github.com/vernal96/go-cms/core"
)

type Module struct{}

func (m *Module) Boot(ctx context.Context, moduleContext core.ModuleContext) error {
	fmt.Println("test module booted")

	app := moduleContext.App()
	runtime := moduleContext.Runtime()

	_ = app
	_ = runtime

	return nil
}

func New() *Module {
	return &Module{}
}

func (m *Module) Code() string {
	return "test"
}

func (m *Module) Register(registry core.Registry) error {
	fmt.Println("test module registered")
	return nil
}

func (m *Module) Requirements() core.ModuleRequirements {
	return core.ModuleRequirements{
		CacheStores: []core.CacheStoreName{"missing_cache"},
		FileDisks:   []core.FileDisk{"missing_disk"},
	}
}
