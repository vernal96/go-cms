package configs

import (
	"github.com/vernal96/go-cms/adapters/cache/memorycache"
	"github.com/vernal96/go-cms/adapters/eventbus/memoryeventbus"
	"github.com/vernal96/go-cms/adapters/storage/memorystorage"
	"github.com/vernal96/go-cms/internal/project"
	"github.com/vernal96/go-cms/internal/testsite"
)

func Dev() project.Config {
	return project.Config{
		CacheStores: []project.CacheStoreRegistration{
			{
				Name:  memorycache.StoreName,
				Store: memorycache.NewStore(),
			},
		},
		FileDisks: []project.FileDiskRegistration{
			{
				Name:    memorystorage.DiskName,
				Storage: memorystorage.NewStorage(),
			},
		},
		SiteProfiles: []project.SiteProfileRegistration{
			{
				Profile: testsite.New(),
			},
		},
		Events: memoryeventbus.NewBus(),
	}
}
