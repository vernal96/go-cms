package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

const (
	CacheStoreDefault = "default"
	CacheStoreRedis   = "redis"
	CacheStoreFile    = "file"
	CacheStoreMemory  = "memory"
	CacheStoreNull    = "null"
)

type CacheStore interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

type CacheManager interface {
	Store(name string) (CacheStore, error)
	Default() (CacheStore, error)
	Redis() (CacheStore, error)
	File() (CacheStore, error)
	Memory() (CacheStore, error)
	Null() CacheStore
}

type DefaultCacheManager struct {
	mu     sync.RWMutex
	stores map[string]CacheStore
}

func NewDefaultCacheManager() *DefaultCacheManager {
	nullStore := NullCacheStore{}

	return &DefaultCacheManager{
		stores: map[string]CacheStore{
			CacheStoreDefault: nullStore,
			CacheStoreNull:    nullStore,
		},
	}
}

func (m *DefaultCacheManager) Store(name string) (CacheStore, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	store, exists := m.stores[name]
	if !exists {
		return nil, fmt.Errorf("cache store %q is not registered", name)
	}

	return store, nil
}

func (m *DefaultCacheManager) Default() (CacheStore, error) {
	return m.Store(CacheStoreDefault)
}

func (m *DefaultCacheManager) Redis() (CacheStore, error) {
	return m.Store(CacheStoreRedis)
}

func (m *DefaultCacheManager) File() (CacheStore, error) {
	return m.Store(CacheStoreFile)
}

func (m *DefaultCacheManager) Memory() (CacheStore, error) {
	return m.Store(CacheStoreMemory)
}

func (m *DefaultCacheManager) Null() CacheStore {
	store, err := m.Store(CacheStoreNull)
	if err != nil {
		return NullCacheStore{}
	}

	return store
}

func (m *DefaultCacheManager) RegisterStore(name string, store CacheStore) error {
	if name == "" {
		return fmt.Errorf("store name is empty")
	}

	if name == CacheStoreDefault {
		return errors.New("use SetDefaultStore instead of registering default store directly")
	}

	if store == nil {
		return errors.New("cache store is nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.stores[name]; exists {
		return fmt.Errorf("cache store %q is already registered", name)
	}

	m.stores[name] = store

	return nil
}

func (m *DefaultCacheManager) SetDefaultStore(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	store, exists := m.stores[name]

	if !exists {
		return fmt.Errorf("cache store %q is not registered", name)
	}

	m.stores[CacheStoreDefault] = store

	return nil
}

type NullCacheStore struct{}

func (s NullCacheStore) Get(ctx context.Context, key string) (any, bool, error) {
	return nil, false, nil
}

func (s NullCacheStore) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return nil
}

func (s NullCacheStore) Delete(ctx context.Context, key string) error {
	return nil
}

func (s NullCacheStore) Clear(ctx context.Context) error {
	return nil
}
