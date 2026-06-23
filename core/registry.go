package core

type Registry interface {
	ForModule(moduleCode string) Registry

	Widgets() WidgetRegistry
	WidgetTemplates() WidgetTemplateRegistry
	Controllers() ControllerRegistry
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
