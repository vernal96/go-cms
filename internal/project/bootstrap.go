package project

import (
	"github.com/vernal96/go-cms/core"
)

func BootstrapApp(config InfrastructureConfig) (*core.App, error) {
	// Создаем менеджер кэша
	cache, err := NewCacheManager(config.CacheStores)
	if err != nil {
		return nil, err
	}

	// Создаем файловый менеджер
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
