package project

import (
	"github.com/vernal96/go-cms/adapters/cache/memorycache"
	"github.com/vernal96/go-cms/adapters/eventbus/memoryeventbus"
	"github.com/vernal96/go-cms/adapters/storage/memorystorage"
	"github.com/vernal96/go-cms/core"
)

const FileDiskMemory core.FileDisk = "memory"

func DevConfig() Config {
	return Config{
		CacheStores: []CacheStoreRegistration{
			{
				Name:    core.CacheStoreMemory,
				Store:   memorycache.NewStore(),
				Default: true,
			},
		},
		FileDisks: []FileDiskRegistration{
			{
				Name:    FileDiskMemory,
				Storage: memorystorage.NewStorage(),
				Default: true,
			},
		},
		Events: memoryeventbus.NewBus(),
	}
}
