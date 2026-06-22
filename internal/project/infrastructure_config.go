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

type InfrastructureConfig struct {
	CacheStores []CacheStoreRegistration
	CacheScopes []CacheScopeRegistration
	FileDisks   []FileDiskRegistration
	Events      core.EventBus
}
