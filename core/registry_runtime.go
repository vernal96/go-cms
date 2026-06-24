package core

type RuntimeRegistry struct {
	moduleCode string
	state      *runtimeRegistryState
}

type runtimeRegistryState struct {
	resourceTypes     map[ResourceType]ResourceTypeDefinition
	resourceTemplates map[ResourceType]map[ResourceTemplateCode]ResourceTemplateDefinition
	resourceFields    map[ResourceType]map[ResourceTemplateCode]map[ResourceFieldCode]ResourceFieldDefinition
	widgets           map[WidgetCode]Widget
	widgetTemplates   map[WidgetCode]map[WidgetTemplateCode]WidgetTemplate
	controllers       []Controller
	controllerRoutes  map[string]struct{}
}

func NewRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{
		state: &runtimeRegistryState{
			resourceTypes:     make(map[ResourceType]ResourceTypeDefinition),
			resourceTemplates: make(map[ResourceType]map[ResourceTemplateCode]ResourceTemplateDefinition),
			resourceFields:    make(map[ResourceType]map[ResourceTemplateCode]map[ResourceFieldCode]ResourceFieldDefinition),
			widgets:           make(map[WidgetCode]Widget),
			widgetTemplates:   make(map[WidgetCode]map[WidgetTemplateCode]WidgetTemplate),
			controllers:       make([]Controller, 0),
			controllerRoutes:  make(map[string]struct{}),
		},
	}
}

func (r *RuntimeRegistry) ForModule(moduleCode string) Registry {
	return &RuntimeRegistry{
		moduleCode: moduleCode,
		state:      r.state,
	}
}

func (r *RuntimeRegistry) ResourceTypes() ResourceTypeRegistry {
	return &runtimeResourceTypeRegistry{
		resourceTypes: r.state.resourceTypes,
	}
}

func (r *RuntimeRegistry) ResourceTemplates() ResourceTemplateRegistry {
	return &runtimeResourceTemplateRegistry{
		resourceTypes:     r.state.resourceTypes,
		resourceTemplates: r.state.resourceTemplates,
	}
}

func (r *RuntimeRegistry) ResourceFields() ResourceFieldRegistry {
	return &runtimeResourceFieldRegistry{
		resourceTypes:     r.state.resourceTypes,
		resourceTemplates: r.state.resourceTemplates,
		resourceFields:    r.state.resourceFields,
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
