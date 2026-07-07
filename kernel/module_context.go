package kernel

type ModuleContext struct {
	app     *App
	runtime *SiteRuntime
	config  any
}

func NewModuleContext(app *App, runtime *SiteRuntime, config any) ModuleContext {
	return ModuleContext{
		app:     app,
		runtime: runtime,
		config:  config,
	}
}

func (c ModuleContext) App() *App {
	return c.app
}

func (c ModuleContext) Runtime() *SiteRuntime {
	return c.runtime
}

func (c ModuleContext) Config() any {
	return c.config
}
