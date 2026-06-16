package core

type App struct {
	cache   CacheManager
	storage FileStorageManager
}

func NewApp() *App {
	return &App{
		cache:   NewDefaultCacheManager(),
		storage: NewDefaultFileStorageManager(),
	}
}

func (a *App) CacheManager() CacheManager {
	return a.cache
}

func (a *App) Storage() FileStorageManager {
	return a.storage
}
