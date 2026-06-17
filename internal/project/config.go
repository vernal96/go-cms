package project

import "github.com/vernal96/go-cms/core"

type CacheStoreRegistration struct {
	Name    string
	Store   core.CacheStore
	Default bool
}

type FileDiskRegistration struct {
	Name    core.FileDisk
	Storage core.FileStorage
	Default bool
}

type Config struct {
	CacheStores []CacheStoreRegistration
	FileDisks   []FileDiskRegistration
	Events      core.EventBus
}
