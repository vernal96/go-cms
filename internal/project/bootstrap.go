package project

import (
	"github.com/vernal96/go-cms/core"
)

func BootstrapApp(config Config) (*core.App, error) {
	cache, err := NewCacheManager(config.CacheStores)
	if err != nil {
		return nil, err
	}

	storage, err := NewFileStorageManager(config.FileDisks)
	if err != nil {
		return nil, err
	}

	app, err := core.NewApp(
		cache,
		storage,
		config.Events,
	)
	if err != nil {
		return nil, err
	}

	return app, nil
}
