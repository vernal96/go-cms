package core

import "errors"

type App struct {
	cache   CacheManager
	storage FileStorageManager
	events  EventBus
}

func NewApp(
	cache CacheManager,
	storage FileStorageManager,
	events EventBus,
) (*App, error) {
	if cache == nil {
		return nil, errors.New("cache manager is nil")
	}

	if storage == nil {
		return nil, errors.New("file storage manager is nil")
	}

	if events == nil {
		return nil, errors.New("event bus is nil")
	}

	return &App{
		cache:   cache,
		storage: storage,
		events:  events,
	}, nil
}

func (a *App) CacheManager() CacheManager {
	return a.cache
}

func (a *App) Storage() FileStorageManager {
	return a.storage
}

func (a *App) EventBus() EventBus {
	return a.events
}
