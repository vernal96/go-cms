package core

import (
	"context"
	"fmt"
)

type App struct {
	registry Registry
	modules  []Module
}

func NewApp(registry Registry) *App {
	if registry == nil {
		registry = NewDefaultRegistry()
	}

	return &App{
		registry: registry,
		modules:  make([]Module, 0),
	}
}

func (a *App) Registry() Registry {
	return a.registry
}

func (a *App) RegisterModule(module Module) error {
	if err := a.registry.RegisterModule(module); err != nil {
		return err
	}

	if err := module.Register(a.registry); err != nil {
		return fmt.Errorf("register module %q: %w", module.Code(), err)
	}

	a.modules = append(a.modules, module)

	return nil
}

func (a *App) Boot(ctx context.Context) error {
	moduleContext := NewModuleContext(a, nil)

	for _, module := range a.modules {
		if err := module.Boot(ctx, moduleContext); err != nil {
			return fmt.Errorf("boot module %q: %w", module.Code(), err)
		}
	}

	return nil
}
