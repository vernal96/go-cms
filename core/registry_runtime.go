package core

type RuntimeRegistry struct {
	moduleCode      string
	widgets         map[WidgetCode]Widget
	widgetTemplates map[WidgetCode]map[WidgetTemplateCode]WidgetTemplate
}

func (r *RuntimeRegistry) Controllers() ControllerRegistry {
	//TODO implement me
	panic("implement me")
}

func NewRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{
		widgets:         make(map[WidgetCode]Widget),
		widgetTemplates: make(map[WidgetCode]map[WidgetTemplateCode]WidgetTemplate),
	}
}

func (r *RuntimeRegistry) ForModule(moduleCode string) Registry {
	return &RuntimeRegistry{
		moduleCode:      moduleCode,
		widgets:         r.widgets,
		widgetTemplates: r.widgetTemplates,
	}
}

func (r *RuntimeRegistry) Widgets() WidgetRegistry {
	return &runtimeWidgetRegistry{
		moduleCode: r.moduleCode,
		widgets:    r.widgets,
	}
}

func (r *RuntimeRegistry) WidgetTemplates() WidgetTemplateRegistry {
	return &runtimeWidgetTemplateRegistry{
		moduleCode:      r.moduleCode,
		widgets:         r.widgets,
		widgetTemplates: r.widgetTemplates,
	}
}

var _ Registry = (*RuntimeRegistry)(nil)
