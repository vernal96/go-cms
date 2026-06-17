package core

type AppDeps struct {
	Cache   CacheManager
	Storage FileStorageManager
	Events  EventBus
}

type App struct {
	cache   CacheManager
	storage FileStorageManager
	events  EventBus
}

func NewApp(deps AppDeps) *App {
	cache := deps.Cache
	if cache == nil {
		cache = NullCacheManager{}
	}

	storage := deps.Storage
	if storage == nil {
		storage = NewDefaultFileStorageManager()
	}

	events := deps.Events
	if events == nil {
		events = NullEventBus{}
	}

	return &App{
		cache:   cache,
		storage: storage,
		events:  events,
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
