package project

import "github.com/vernal96/go-cms/core"

type CacheStoreRegistration struct {
	Name  core.CacheStoreName
	Store core.CacheStore
}

type CacheScopeRegistration struct {
	Scope core.CacheScope
	Store core.CacheStore
}

type FileDiskRegistration struct {
	Name    core.FileDisk
	Storage core.FileStorage
}

type InfrastructureRegistry struct {
	cacheStores []CacheStoreRegistration
	cacheScopes []CacheScopeRegistration
	fileDisks   []FileDiskRegistration
	events      core.EventBus
}

func NewInfrastructureRegistry() *InfrastructureRegistry {
	return &InfrastructureRegistry{
		events: core.NullEventBus{},
	}
}

func (r *InfrastructureRegistry) RegisterCacheStore(name core.CacheStoreName, store core.CacheStore) {
	r.cacheStores = append(r.cacheStores, CacheStoreRegistration{
		Name:  name,
		Store: store,
	})
}

func (r *InfrastructureRegistry) RegisterCacheScope(scope core.CacheScope, store core.CacheStore) {
	r.cacheScopes = append(r.cacheScopes, CacheScopeRegistration{
		Scope: scope,
		Store: store,
	})
}

func (r *InfrastructureRegistry) RegisterFileDisk(name core.FileDisk, storage core.FileStorage) {
	r.fileDisks = append(r.fileDisks, FileDiskRegistration{
		Name:    name,
		Storage: storage,
	})
}

func (r *InfrastructureRegistry) UseEventBus(events core.EventBus) {
	r.events = events
}

func (r *InfrastructureRegistry) CacheStores() []CacheStoreRegistration {
	return r.cacheStores
}

func (r *InfrastructureRegistry) CacheScopes() []CacheScopeRegistration {
	return r.cacheScopes
}

func (r *InfrastructureRegistry) FileDisks() []FileDiskRegistration {
	return r.fileDisks
}

func (r *InfrastructureRegistry) EventBus() core.EventBus {
	return r.events
}
