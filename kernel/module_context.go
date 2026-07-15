package kernel

type ModuleContext struct {
	app          *App
	runtime      *SiteRuntime
	moduleConfig any
}

func NewModuleContext(app *App, runtime *SiteRuntime, moduleConfig any) ModuleContext {
	return ModuleContext{
		app:          app,
		runtime:      runtime,
		moduleConfig: moduleConfig,
	}
}

func (c ModuleContext) App() *App {
	return c.app
}

func (c ModuleContext) Runtime() *SiteRuntime {
	return c.runtime
}

func (c ModuleContext) ModuleConfig() any {
	return c.moduleConfig
}
