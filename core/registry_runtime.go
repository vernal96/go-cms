package core

type RuntimeRegistry struct {
	moduleCode string
	state      *runtimeRegistryState
}

type runtimeRegistryState struct {
	widgets         map[WidgetCode]Widget
	widgetTemplates map[WidgetCode]map[WidgetTemplateCode]WidgetTemplate
	controllers     []Controller
	controllerRoutes map[string]struct{}
}

func NewRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{
		state: &runtimeRegistryState{
			widgets:          make(map[WidgetCode]Widget),
			widgetTemplates:  make(map[WidgetCode]map[WidgetTemplateCode]WidgetTemplate),
			controllers:      make([]Controller, 0),
			controllerRoutes: make(map[string]struct{}),
		},
	}
}

func (r *RuntimeRegistry) ForModule(moduleCode string) Registry {
	return &RuntimeRegistry{
		moduleCode: moduleCode,
		state:      r.state,
	}
}

func (r *RuntimeRegistry) Widgets() WidgetRegistry {
	return &runtimeWidgetRegistry{
		moduleCode: r.moduleCode,
		widgets:    r.state.widgets,
	}
}

func (r *RuntimeRegistry) WidgetTemplates() WidgetTemplateRegistry {
	return &runtimeWidgetTemplateRegistry{
		moduleCode:      r.moduleCode,
		widgets:         r.state.widgets,
		widgetTemplates: r.state.widgetTemplates,
	}
}

func (r *RuntimeRegistry) Controllers() ControllerRegistry {
	return &runtimeControllerRegistry{
		controllers:      &r.state.controllers,
		controllerRoutes: r.state.controllerRoutes,
	}
}

var _ Registry = (*RuntimeRegistry)(nil)
