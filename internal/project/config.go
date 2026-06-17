package project

import "github.com/vernal96/go-cms/core"

type CacheStoreRegistration struct {
	Name  core.CacheStoreName
	Store core.CacheStore
}

type FileDiskRegistration struct {
	Name    core.FileDisk
	Storage core.FileStorage
}

type Config struct {
	CacheStores []CacheStoreRegistration
	FileDisks   []FileDiskRegistration
	Events      core.EventBus
}
