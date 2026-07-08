package kernel

type ModuleContext struct {
	app             *App
	runtime         *SiteRuntime
	moduleConfig    any
	adapterDefaults AdapterDefaults
}

func NewModuleContext(
	app *App,
	runtime *SiteRuntime,
	moduleConfig any,
	adapterDefaults AdapterDefaults,
) ModuleContext {
	return ModuleContext{
		app:             app,
		runtime:         runtime,
		moduleConfig:    moduleConfig,
		adapterDefaults: adapterDefaults,
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

func (c ModuleContext) AdapterDefaults() AdapterDefaults {
	return c.adapterDefaults
}
