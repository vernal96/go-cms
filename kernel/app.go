package kernel

import (
	"context"
	"fmt"
)

type AppConfig struct {
	dbConnector DBConnector
}

type App struct {
	config          AppConfig
	adapters        AdapterRegistry
	profiles        map[ProfileCode]Profile
	profileRuntimes map[ProfileCode]*ProfileRuntime
}

func NewApp(
	ctx context.Context,
	config AppConfig,
	adapters AdapterRegistry,
	profiles []Profile,
) (*App, error) {
	if ctx == nil {
		return nil, fmt.Errorf("app context is nil")
	}

	if adapters == nil {
		adapters = NewAdapterRegistry()
	}

	app := &App{
		config:          config,
		adapters:        adapters,
		profiles:        make(map[ProfileCode]Profile),
		profileRuntimes: make(map[ProfileCode]*ProfileRuntime),
	}

	for _, profile := range profiles {
		if err := app.registerProfile(profile); err != nil {
			return nil, err
		}
	}

	runtimeFactory, err := NewProfileRuntimeFactory(app)
	if err != nil {
		return nil, err
	}

	for _, profile := range profiles {
		runtime, err := runtimeFactory.Make(ctx, profile)
		if err != nil {
			return nil, fmt.Errorf(
				"build profile runtime %q: %w",
				profile.Code(),
				err,
			)
		}

		app.profileRuntimes[profile.Code()] = runtime
	}

	return app, nil
}

func (a *App) registerProfile(profile Profile) error {
	if profile == nil {
		return fmt.Errorf("profile is nil")
	}

	code := profile.Code()
	if code == "" {
		return fmt.Errorf("profile code is empty")
	}

	if _, exists := a.profiles[code]; exists {
		return fmt.Errorf(
			"profile %q already registered",
			code,
		)
	}

	a.profiles[code] = profile

	return nil
}

func (a *App) Config() AppConfig {
	return a.config
}

func (a *App) Adapters() AdapterRegistry {
	return a.adapters
}

func (a *App) Profile(
	code ProfileCode,
) (Profile, bool) {
	profile, exists := a.profiles[code]
	return profile, exists
}

func (a *App) ProfileRuntime(
	code ProfileCode,
) (*ProfileRuntime, bool) {
	runtime, exists := a.profileRuntimes[code]
	return runtime, exists
}
