package core

import (
	"context"
	"fmt"
)

type SiteRuntimeFactory struct {
	app *App
}

func NewSiteRuntimeFactory(app *App) *SiteRuntimeFactory {
	return &SiteRuntimeFactory{app: app}
}

func (f *SiteRuntimeFactory) Make(ctx context.Context, profile Profile) (*SiteRuntime, error) {
	registry := NewRuntimeRegistry()

	for _, module := range profile.Modules() {
		moduleRegistry := registry.ForModule(module.Code())

		if err := module.Register(moduleRegistry); err != nil {
			return nil, fmt.Errorf("register module %q: %w", module.Code(), err)
		}
	}

	runtime := NewSiteRuntime(f.app, profile, registry)
	moduleContext := NewModuleContext(f.app)

	for _, module := range profile.Modules() {
		if err := module.Boot(ctx, moduleContext); err != nil {
			return nil, fmt.Errorf("boot module %q: %w", module.Code(), err)
		}
	}

	return runtime, nil
}
