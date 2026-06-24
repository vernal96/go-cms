package project

import (
	"github.com/vernal96/go-cms/core"
)

func BootstrapApp(infrastructure *InfrastructureRegistry) (*core.App, error) {
	cache, err := NewCacheManager(
		infrastructure.CacheStores(),
		infrastructure.CacheScopes(),
	)
	if err != nil {
		return nil, err
	}

	storage, err := NewFileStorageManager(infrastructure.FileDisks())
	if err != nil {
		return nil, err
	}

	app, err := core.NewApp(
		cache,
		storage,
		infrastructure.EventBus(),
		infrastructure.Logger(),
		infrastructure.ResourceRepository(),
	)
	if err != nil {
		return nil, err
	}

	return app, nil
}
