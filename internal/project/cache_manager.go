package project

import (
	"errors"
	"fmt"

	"github.com/vernal96/go-cms/core"
)

type CacheManager struct {
	stores map[core.CacheStoreName]core.CacheStore
	scopes map[core.CacheScope]core.CacheStore
}

func NewCacheManager(
	storeRegistrations []CacheStoreRegistration,
	scopeRegistrations []CacheScopeRegistration,
) (*CacheManager, error) {
	stores := make(map[core.CacheStoreName]core.CacheStore, len(storeRegistrations))

	for _, registration := range storeRegistrations {
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

	scopes := make(map[core.CacheScope]core.CacheStore, len(scopeRegistrations))

	for _, registration := range scopeRegistrations {
		if registration.Scope == "" {
			return nil, errors.New("cache scope is empty")
		}

		if registration.Store == nil {
			return nil, fmt.Errorf("cache scope %q store is nil", registration.Scope)
		}

		if _, exists := scopes[registration.Scope]; exists {
			return nil, fmt.Errorf("cache scope %q is already registered", registration.Scope)
		}

		scopes[registration.Scope] = registration.Store
	}

	return &CacheManager{
		stores: stores,
		scopes: scopes,
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

func (m *CacheManager) Scope(scope core.CacheScope) (core.CacheStore, error) {
	if scope == "" {
		return nil, errors.New("cache scope is empty")
	}

	store, exists := m.scopes[scope]
	if !exists {
		return nil, fmt.Errorf("cache scope %q is not registered", scope)
	}

	return store, nil
}

var _ core.CacheManager = (*CacheManager)(nil)
