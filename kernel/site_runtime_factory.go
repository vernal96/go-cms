package kernel

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

	modules := profile.Modules()

	if err := f.validateModules(profile, modules); err != nil {
		return nil, err
	}

	for _, module := range modules {
		moduleRegistry := registry.ForModule(module.Code())

		if err := module.Register(moduleRegistry); err != nil {
			return nil, fmt.Errorf("register module %q: %w", module.Code(), err)
		}
	}

	runtime := NewSiteRuntime(f.app, profile, registry)
	moduleContext := NewModuleContext(f.app)

	for _, module := range modules {
		if err := module.Boot(ctx, moduleContext); err != nil {
			return nil, fmt.Errorf("boot module %q: %w", module.Code(), err)
		}
	}

	return runtime, nil
}

func (f *SiteRuntimeFactory) validateModules(profile Profile, modules []Module) error {
	if profile == nil {
		return fmt.Errorf("profile is nil")
	}

	if len(modules) == 0 {
		return fmt.Errorf("profile %q has no modules", profile.Code())
	}

	seen := make(map[ModuleCode]struct{}, len(modules))

	for index, module := range modules {
		if module == nil {
			return fmt.Errorf("profile %q has nil module at index %d", profile.Code(), index)
		}

		moduleCode := module.Code()
		if moduleCode == "" {
			return fmt.Errorf("profile %q has module with empty code at index %d", profile.Code(), index)
		}

		if _, exists := seen[moduleCode]; exists {
			return fmt.Errorf("profile %q has duplicate module %q", profile.Code(), moduleCode)
		}

		seen[moduleCode] = struct{}{}
	}

	return nil
}
