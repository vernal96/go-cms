package kernel

import (
	"context"
	"fmt"
)

type ProfileRuntimeFactory struct {
	app *App
}

func NewProfileRuntimeFactory(
	app *App,
) (*ProfileRuntimeFactory, error) {
	if app == nil {
		return nil, fmt.Errorf("app is nil")
	}

	return &ProfileRuntimeFactory{
		app: app,
	}, nil
}

func (f *ProfileRuntimeFactory) Make(
	ctx context.Context,
	profile Profile,
) (*ProfileRuntime, error) {
	if ctx == nil {
		return nil, fmt.Errorf("profile runtime context is nil")
	}

	if profile == nil {
		return nil, fmt.Errorf("profile is nil")
	}

	if profile.Code() == "" {
		return nil, fmt.Errorf("profile code is empty")
	}

	profileModules := profile.Modules()
	moduleCodes := make(
		map[ModuleCode]struct{},
		len(profileModules),
	)

	for index, profileModule := range profileModules {
		module := profileModule.Module
		if module == nil {
			return nil, fmt.Errorf(
				"profile %q module at index %d is nil",
				profile.Code(),
				index,
			)
		}

		moduleCode := module.Code()
		if moduleCode == "" {
			return nil, fmt.Errorf(
				"profile %q module at index %d has empty code",
				profile.Code(),
				index,
			)
		}

		if _, exists := moduleCodes[moduleCode]; exists {
			return nil, fmt.Errorf(
				"profile %q contains duplicate module %q",
				profile.Code(),
				moduleCode,
			)
		}

		moduleCodes[moduleCode] = struct{}{}
	}

	registry := NewRuntimeRegistry()

	for _, profileModule := range profileModules {
		module := profileModule.Module
		moduleRegistry := registry.ForModule(module.Code())

		if err := module.Register(moduleRegistry); err != nil {
			return nil, fmt.Errorf(
				"register module %q: %w",
				module.Code(),
				err,
			)
		}
	}

	runtime := NewProfileRuntime(
		f.app,
		profile,
		registry,
	)

	for _, profileModule := range profileModules {
		module := profileModule.Module

		moduleContext := NewModuleContext(
			f.app,
			runtime,
			profileModule.ModuleConfig,
		)

		if err := module.Boot(ctx, moduleContext); err != nil {
			return nil, fmt.Errorf(
				"boot module %q: %w",
				module.Code(),
				err,
			)
		}
	}

	return runtime, nil
}
