package core

type ModuleContext struct {
	app     *App
	runtime *SiteRuntime
}

func NewModuleContext(app *App, runtime *SiteRuntime) ModuleContext {
	return ModuleContext{
		app:     app,
		runtime: runtime,
	}
}

func (c ModuleContext) App() *App {
	return c.app
}

func (c ModuleContext) Runtime() *SiteRuntime {
	return c.runtime
}
