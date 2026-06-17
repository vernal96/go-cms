package project

import (
	"github.com/vernal96/go-cms/adapters/cache/memorycache"
	"github.com/vernal96/go-cms/adapters/eventbus/memoryeventbus"
	"github.com/vernal96/go-cms/adapters/storage/memorystorage"
)

func DevConfig() Config {
	return Config{
		CacheStores: []CacheStoreRegistration{
			{
				Name:  memorycache.StoreName,
				Store: memorycache.NewStore(),
			},
		},
		FileDisks: []FileDiskRegistration{
			{
				Name:    memorystorage.DiskName,
				Storage: memorystorage.NewStorage(),
			},
		},
		Events: memoryeventbus.NewBus(),
	}
}
