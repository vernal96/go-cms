package kernel

type ModuleContext struct {
	app          *App
	runtime      *ProfileRuntime
	moduleConfig any
}

func NewModuleContext(
	app *App,
	runtime *ProfileRuntime,
	moduleConfig any,
) ModuleContext {
	return ModuleContext{
		app:          app,
		runtime:      runtime,
		moduleConfig: moduleConfig,
	}
}

func (c ModuleContext) App() *App {
	return c.app
}

func (c ModuleContext) Runtime() *ProfileRuntime {
	return c.runtime
}

func (c ModuleContext) ModuleConfig() any {
	return c.moduleConfig
}
