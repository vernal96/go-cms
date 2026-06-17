package project

import (
	"github.com/vernal96/go-cms/adapters/cache/memorycache"
	"github.com/vernal96/go-cms/adapters/eventbus/memoryeventbus"
	"github.com/vernal96/go-cms/adapters/storage/memorystorage"
	"github.com/vernal96/go-cms/core"
)

const FileDiskMemory core.FileDisk = "memory"

func BootstrapApp() (*core.App, error) {
	cache := core.NewDefaultCacheManager()

	if err := cache.RegisterStore(core.CacheStoreMemory, memorycache.NewStore()); err != nil {
		return nil, err
	}

	if err := cache.SetDefaultStore(core.CacheStoreMemory); err != nil {
		return nil, err
	}

	storage := core.NewDefaultFileStorageManager()

	if err := storage.RegisterDisk(FileDiskMemory, memorystorage.NewStorage()); err != nil {
		return nil, err
	}

	if err := storage.SetDefaultDisk(FileDiskMemory); err != nil {
		return nil, err
	}

	events := memoryeventbus.NewBus()

	app := core.NewApp(core.AppDeps{
		Cache:   cache,
		Storage: storage,
		Events:  events,
	})

	return app, nil
}
