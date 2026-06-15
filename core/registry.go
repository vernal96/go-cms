package core

import (
	"errors"
	"fmt"
)

type Registry interface {
	RegisterModule(module Module) error
	Module(code string) (Module, bool)
}

type DefaultRegistry struct {
	modules map[string]Module
}

func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{
		modules: make(map[string]Module),
	}
}

func (r *DefaultRegistry) RegisterModule(module Module) error {
	if module == nil {
		return errors.New("module is nil")
	}

	code := module.Code()

	if code == "" {
		return errors.New("module code is empty")
	}

	if _, exists := r.modules[code]; exists {
		return fmt.Errorf("module %q already registered", code)
	}

	r.modules[code] = module

	return nil
}

func (r *DefaultRegistry) Module(code string) (Module, bool) {
	module, exists := r.modules[code]
	return module, exists
}
