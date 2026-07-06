package kernel

type ModuleContext struct {
	app *App
}

func NewModuleContext(app *App) ModuleContext {
	return ModuleContext{app: app}
}

func (c ModuleContext) App() *App {
	return c.app
}
