package kernel

type App struct {
	profiles         []Profile
	connectorManager *ConnectorManager
}

func NewApp(
	profiles []Profile,
	connectorManager *ConnectorManager,
) *App {

	return &App{
		profiles:         profiles,
		connectorManager: connectorManager,
	}
}

func (app *App) Profiles() []Profile {
	return app.profiles
}

func (app *App) ConnectorManager() *ConnectorManager {
	return app.connectorManager
}
