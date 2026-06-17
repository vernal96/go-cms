package project

import (
	"errors"
	"fmt"

	"github.com/vernal96/go-cms/core"
)

type CacheManager struct {
	stores map[core.CacheStoreName]core.CacheStore
}

func NewCacheManager(registrations []CacheStoreRegistration) (*CacheManager, error) {
	stores := make(map[core.CacheStoreName]core.CacheStore, len(registrations))

	for _, registration := range registrations {
		if registration.Name == "" {
			return nil, errors.New("cache store name is empty")
		}

		if registration.Store == nil {
			return nil, fmt.Errorf("cache store %q is nil", registration.Name)
		}

		if _, exists := stores[registration.Name]; exists {
			return nil, fmt.Errorf("cache store %q is already registered", registration.Name)
		}

		stores[registration.Name] = registration.Store
	}

	return &CacheManager{
		stores: stores,
	}, nil
}

func (m *CacheManager) Store(name core.CacheStoreName) (core.CacheStore, error) {
	if name == "" {
		return nil, errors.New("cache store name is empty")
	}

	store, exists := m.stores[name]
	if !exists {
		return nil, fmt.Errorf("cache store %q is not registered", name)
	}

	return store, nil
}

var _ core.CacheManager = (*CacheManager)(nil)
