package configs

import (
	"github.com/vernal96/go-cms/adapters/cache/memorycache"
	"github.com/vernal96/go-cms/adapters/eventbus/memoryeventbus"
	"github.com/vernal96/go-cms/adapters/storage/memorystorage"
	"github.com/vernal96/go-cms/internal/project"
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
		Events: memoryeventbus.NewBus(),
	}
}
