package kernel

type AppConfig struct {
	AdapterDefaults AdapterDefaults
}

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

func (a *App) AdapterDefaults() AdapterDefaults {
	return ResolveAdapterDefaults(a.config.AdapterDefaults)
}

func (a *App) Adapters() AdapterRegistry {
	return a.adapters
}
