package core

type AppConfig struct {
}

type App struct{}

func NewApp(config AppConfig) *App {
	return &App{}
}
