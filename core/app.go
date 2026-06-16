package core

type App struct {
	cache CacheManager
}

func NewApp() *App {
	return &App{
		cache: NewDefaultCacheManager(),
	}
}

func (a *App) CacheManager() CacheManager {
	return a.cache
}
