package core

import (
	"context"
	"errors"
	"fmt"
)

type SiteRuntimeFactory struct {
	app      *App
	profiles SiteProfileRegistry
}

func NewSiteRuntimeFactory(app *App, profiles SiteProfileRegistry) *SiteRuntimeFactory {
	return &SiteRuntimeFactory{
		app:      app,
		profiles: profiles,
	}
}

func (f *SiteRuntimeFactory) Make(ctx context.Context, site Site) (*SiteRuntime, error) {
	if f.app == nil {
		return nil, errors.New("app is nil")
	}

	if f.profiles == nil {
		return nil, errors.New("site profile registry is nil")
	}

	if site.ProfileCode == "" {
		return nil, errors.New("site profile code is empty")
	}

	profile, exists := f.profiles.Profile(site.ProfileCode)
	if !exists {
		return nil, fmt.Errorf("site profile %q not found", site.ProfileCode)
	}

	registry := NewDefaultRegistry()
	modules := profile.Modules()

	seenModules := make(map[string]struct{}, len(modules))

	for _, module := range modules {
		if module == nil {
			return nil, errors.New("site module is nil")
		}

		code := module.Code()
		if code == "" {
			return nil, errors.New("site module code is empty")
		}

		if _, exists := seenModules[code]; exists {
			return nil, fmt.Errorf("site module %q already registered", code)
		}

		seenModules[code] = struct{}{}

		if err := module.Register(registry); err != nil {
			return nil, fmt.Errorf("register site module extensions %q: %w", code, err)
		}
	}

	runtime, err := NewSiteRuntime(site, profile, registry)
	if err != nil {
		return nil, err
	}

	moduleContext := NewModuleContext(f.app, runtime)

	for _, module := range modules {
		if err := module.Boot(ctx, moduleContext); err != nil {
			return nil, fmt.Errorf("boot site module %q: %w", module.Code(), err)
		}
	}

	return runtime, nil
}
