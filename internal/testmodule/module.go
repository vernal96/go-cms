package testmodule

import (
	"context"
	"fmt"

	"github.com/vernal96/go-cms/core"
)

type Config struct {
	CacheStore core.CacheStoreName
	FileDisk   core.FileDisk
}

type Module struct {
	config Config
}

func New(config Config) *Module {
	return &Module{
		config: config,
	}
}

func (m *Module) Code() string {
	return "test"
}

func (m *Module) Register(registry core.Registry) error {
	fmt.Println("test module registered")
	return nil
}

func (m *Module) Boot(ctx context.Context, moduleContext core.ModuleContext) error {
	fmt.Println("test module booted")

	app := moduleContext.App()
	runtime := moduleContext.Runtime()

	_ = app
	_ = runtime

	return nil
}

func (m *Module) Requirements() core.ModuleRequirements {
	requirements := core.ModuleRequirements{}

	if m.config.CacheStore != "" {
		requirements.CacheStores = append(requirements.CacheStores, m.config.CacheStore)
	}

	if m.config.FileDisk != "" {
		requirements.FileDisks = append(requirements.FileDisks, m.config.FileDisk)
	}

	return requirements
}
