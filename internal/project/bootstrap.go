package project

import (
	"github.com/vernal96/go-cms/core"
)

func BootstrapApp(config Config) (*core.App, error) {
	cache := core.NewDefaultCacheManager()

	for _, store := range config.CacheStores {
		if err := cache.RegisterStore(store.Name, store.Store); err != nil {
			return nil, err
		}

		if store.Default {
			if err := cache.SetDefaultStore(store.Name); err != nil {
				return nil, err
			}
		}
	}

	storage := core.NewDefaultFileStorageManager()

	for _, disk := range config.FileDisks {
		if err := storage.RegisterDisk(disk.Name, disk.Storage); err != nil {
			return nil, err
		}

		if disk.Default {
			if err := storage.SetDefaultDisk(disk.Name); err != nil {
				return nil, err
			}
		}
	}

	app := core.NewApp(core.AppDeps{
		Cache:   cache,
		Storage: storage,
		Events:  config.Events,
	})

	return app, nil
}
