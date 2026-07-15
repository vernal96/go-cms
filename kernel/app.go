package kernel

type AppConfig struct{}

type App struct {
	config   AppConfig
	adapters AdapterRegistry
}

func NewApp(config AppConfig) *App {
	return &App{
		config:   config,
		adapters: NewAdapterRegistry(),
	}
}

func (a *App) Config() AppConfig {
	return a.config
}

func (a *App) Adapters() AdapterRegistry {
	return a.adapters
}
