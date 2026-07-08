package kernel

type AppConfig struct {
	AdapterDefaults AdapterDefaults
}

type App struct {
	config AppConfig
}

func NewApp(config AppConfig) *App {
	return &App{
		config: config,
	}
}

func (a *App) Config() AppConfig {
	return a.config
}

func (a *App) AdapterDefaults() AdapterDefaults {
	return ResolveAdapterDefaults(a.config.AdapterDefaults)
}
