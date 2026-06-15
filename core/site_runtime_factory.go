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

	if site.Code == "" {
		return nil, errors.New("site code is empty")
	}

	profile, exists := f.profiles.Profile(site.Code)
	if !exists {
		return nil, fmt.Errorf("site profile %q not found", site.Code)
	}

	registry := NewDefaultRegistry()

	modules := profile.Modules()

	for _, module := range modules {
		if err := registry.RegisterModule(module); err != nil {
			return nil, fmt.Errorf("register site module %q: %w", module.Code(), err)
		}

		if err := module.Register(registry); err != nil {
			return nil, fmt.Errorf("register site module extensions %q: %w", module.Code(), err)
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
