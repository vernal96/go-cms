package kernel

import "fmt"

type AdapterCode string

type AdapterContractCode string

type AdapterRegistry interface {
	Add(contract AdapterContractCode, code AdapterCode, adapter any) error
	Get(contract AdapterContractCode, code AdapterCode) (any, bool)
}

type DefaultAdapterRegistry struct {
	adapters map[AdapterContractCode]map[AdapterCode]any
}

func NewAdapterRegistry() *DefaultAdapterRegistry {
	return &DefaultAdapterRegistry{
		adapters: make(map[AdapterContractCode]map[AdapterCode]any),
	}
}

func (r *DefaultAdapterRegistry) Add(contract AdapterContractCode, code AdapterCode, adapter any) error {
	if contract == "" {
		return fmt.Errorf("adapter contract is empty")
	}

	if code == "" {
		return fmt.Errorf("adapter code is empty")
	}

	if adapter == nil {
		return fmt.Errorf("adapter %q for contract %q is nil", code, contract)
	}

	if _, exists := r.adapters[contract]; !exists {
		r.adapters[contract] = make(map[AdapterCode]any)
	}

	if _, exists := r.adapters[contract][code]; exists {
		return fmt.Errorf("adapter %q for contract %q already registered", code, contract)
	}

	r.adapters[contract][code] = adapter

	return nil
}

func (r *DefaultAdapterRegistry) Get(contract AdapterContractCode, code AdapterCode) (any, bool) {
	contractAdapters, exists := r.adapters[contract]
	if !exists {
		return nil, false
	}

	adapter, exists := contractAdapters[code]
	return adapter, exists
}

func AdapterAs[T any](registry AdapterRegistry, contract AdapterContractCode, code AdapterCode) (T, error) {
	var zero T

	if registry == nil {
		return zero, fmt.Errorf("adapter registry is nil")
	}

	rawAdapter, exists := registry.Get(contract, code)
	if !exists {
		return zero, fmt.Errorf("adapter %q for contract %q is not registered", code, contract)
	}

	adapter, ok := rawAdapter.(T)
	if !ok {
		return zero, fmt.Errorf("adapter %q for contract %q has invalid type", code, contract)
	}

	return adapter, nil
}

var _ AdapterRegistry = (*DefaultAdapterRegistry)(nil)
