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

type SiteProfileRegistration struct {
	Profile core.SiteProfile
}

type Config struct {
	CacheStores  []CacheStoreRegistration
	FileDisks    []FileDiskRegistration
	SiteProfiles []SiteProfileRegistration
	Events       core.EventBus
}
