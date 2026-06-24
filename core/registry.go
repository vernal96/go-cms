package core

type Registry interface {
	ForModule(moduleCode string) Registry

	ResourceTypes() ResourceTypeRegistry
	ResourceTemplates() ResourceTemplateRegistry
	ResourceFields() ResourceFieldRegistry
	ResourceTemplateRenderers() ResourceTemplateRendererRegistry
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

type ResourceFieldRegistry interface {
	Register(field ResourceFieldDefinition) error
	Get(
		resourceType ResourceType,
		template ResourceTemplateCode,
		code ResourceFieldCode,
	) (ResourceFieldDefinition, bool)
	AllForTemplate(
		resourceType ResourceType,
		template ResourceTemplateCode,
	) []ResourceFieldDefinition
}

type ResourceTemplateRendererRegistry interface {
	Register(renderer ResourceTemplateRenderer) error
	Get(
		resourceType ResourceType,
		template ResourceTemplateCode,
	) (ResourceTemplateRenderer, bool)
	AllForResourceType(resourceType ResourceType) []ResourceTemplateRenderer
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
