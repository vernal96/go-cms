package core

type Registry interface {
	ForModule(moduleCode string) Registry

	ResourceTypes() ResourceTypeRegistry
	ResourceTemplates() ResourceTemplateRegistry
	Widgets() WidgetRegistry
	WidgetTemplates() WidgetTemplateRegistry
	Controllers() ControllerRegistry
}

type ResourceTypeRegistry interface {
	Register(resourceType ResourceTypeDefinition) error
	Get(code ResourceType) (ResourceTypeDefinition, bool)
	All() []ResourceTypeDefinition
}

type ResourceTemplateRegistry interface {
	Register(template ResourceTemplateDefinition) error
	Get(resourceType ResourceType, code ResourceTemplateCode) (ResourceTemplateDefinition, bool)
	AllForResourceType(resourceType ResourceType) []ResourceTemplateDefinition
}

type WidgetRegistry interface {
	Register(widget Widget) error
	Get(code WidgetCode) (Widget, bool)
	All() []Widget
}

type WidgetTemplateRegistry interface {
	RegisterForWidget(widget WidgetCode, template WidgetTemplate) error
	Get(widget WidgetCode, template WidgetTemplateCode) (WidgetTemplate, bool)
	AllForWidget(widget WidgetCode) []WidgetTemplate
}

type ControllerRegistry interface {
	Register(controller Controller) error
	All() []Controller
}
