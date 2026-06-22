package configs

import (
	"github.com/vernal96/go-cms/adapters/cache/memorycache"
	"github.com/vernal96/go-cms/adapters/eventbus/memoryeventbus"
	"github.com/vernal96/go-cms/adapters/storage/memorystorage"
	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/project"
	"github.com/vernal96/go-cms/internal/testmodule"
	"github.com/vernal96/go-cms/internal/testsite"
)

func DevInfrastructure() project.InfrastructureConfig {
	cacheStore := memorycache.NewStore()

	return project.InfrastructureConfig{
		CacheStores: []project.CacheStoreRegistration{
			{
				Name:  memorycache.StoreName,
				Store: cacheStore,
			},
		},
		CacheScopes: []project.CacheScopeRegistration{
			{
				Scope: core.CacheScopeDefault,
				Store: cacheStore,
			},
			{
				Scope: testmodule.CacheScopeDefault,
				Store: cacheStore,
			},
		},
		FileDisks: []project.FileDiskRegistration{
			{
				Name:    memorystorage.DiskName,
				Storage: memorystorage.NewStorage(),
			},
		},
		Events: memoryeventbus.NewBus(),
	}
}

func DevSiteProfiles() project.SiteProfileConfig {
	return project.SiteProfileConfig{
		Profiles: []project.SiteProfileRegistration{
			{
				Profile: testsite.New(),
			},
		},
	}
}
