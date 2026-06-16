package core

import "errors"

type App struct {
	cache   CacheManager
	storage FileStorageManager
	events  EventBus
}

func NewApp() *App {
	return &App{
		cache:   NewDefaultCacheManager(),
		storage: NewDefaultFileStorageManager(),
		events:  NullEventBus{},
	}
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

func (a *App) SetEventBus(bus EventBus) error {
	if bus == nil {
		return errors.New("event bus is nil")
	}

	a.events = bus

	return nil
}
