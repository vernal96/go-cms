package core

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type Registry interface {
	ForModule(moduleCode string) Registry

	Widgets() WidgetRegistry
	WidgetTemplates() WidgetTemplateRegistry
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

type RuntimeRegistry struct {
	moduleCode      string
	widgets         map[WidgetCode]Widget
	widgetTemplates map[WidgetCode]map[WidgetTemplateCode]WidgetTemplate
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

type runtimeWidgetRegistry struct {
	moduleCode string
	widgets    map[WidgetCode]Widget
}

func (r *runtimeWidgetRegistry) Register(widget Widget) error {
	if widget == nil {
		return errors.New("widget is nil")
	}

	code, err := r.fullWidgetCode(widget.Code())
	if err != nil {
		return err
	}

	if _, exists := r.widgets[code]; exists {
		return fmt.Errorf("widget %q is already registered", code)
	}

	r.widgets[code] = widget

	return nil
}

func (r *runtimeWidgetRegistry) Get(code WidgetCode) (Widget, bool) {
	widget, exists := r.widgets[code]

	return widget, exists
}

func (r *runtimeWidgetRegistry) All() []Widget {
	widgets := make([]Widget, 0, len(r.widgets))

	for _, widget := range r.widgets {
		widgets = append(widgets, widget)
	}

	slices.SortFunc(widgets, func(a, b Widget) int {
		return strings.Compare(string(a.Code()), string(b.Code()))
	})

	return widgets
}

func (r *runtimeWidgetRegistry) fullWidgetCode(code WidgetCode) (WidgetCode, error) {
	if code == "" {
		return "", errors.New("widget code is empty")
	}

	if strings.Contains(string(code), ".") {
		return "", fmt.Errorf("widget code %q must be local", code)
	}

	if r.moduleCode == "" {
		return "", errors.New("module code is empty")
	}

	return WidgetCode(r.moduleCode + "." + string(code)), nil
}

type runtimeWidgetTemplateRegistry struct {
	moduleCode string

	widgets         map[WidgetCode]Widget
	widgetTemplates map[WidgetCode]map[WidgetTemplateCode]WidgetTemplate
}

func (r *runtimeWidgetTemplateRegistry) RegisterForWidget(
	widgetCode WidgetCode,
	template WidgetTemplate,
) error {
	if template == nil {
		return errors.New("widget template is nil")
	}

	fullWidgetCode, err := r.fullWidgetCode(widgetCode)
	if err != nil {
		return err
	}

	if _, exists := r.widgets[fullWidgetCode]; !exists {
		return fmt.Errorf("widget %q is not registered", fullWidgetCode)
	}

	templateCode := template.Code()
	if templateCode == "" {
		return errors.New("widget template code is empty")
	}

	if templateCode == WidgetTemplateDefault {
		return errors.New("default widget template must not be registered")
	}

	if _, exists := r.widgetTemplates[fullWidgetCode]; !exists {
		r.widgetTemplates[fullWidgetCode] = make(map[WidgetTemplateCode]WidgetTemplate)
	}

	if _, exists := r.widgetTemplates[fullWidgetCode][templateCode]; exists {
		return fmt.Errorf(
			"widget template %q for widget %q is already registered",
			templateCode,
			fullWidgetCode,
		)
	}

	r.widgetTemplates[fullWidgetCode][templateCode] = template

	return nil
}

func (r *runtimeWidgetTemplateRegistry) Get(
	widgetCode WidgetCode,
	templateCode WidgetTemplateCode,
) (WidgetTemplate, bool) {
	templates, exists := r.widgetTemplates[widgetCode]
	if !exists {
		return nil, false
	}

	template, exists := templates[templateCode]

	return template, exists
}

func (r *runtimeWidgetTemplateRegistry) AllForWidget(widgetCode WidgetCode) []WidgetTemplate {
	templatesMap, exists := r.widgetTemplates[widgetCode]
	if !exists {
		return nil
	}

	templates := make([]WidgetTemplate, 0, len(templatesMap))

	for _, template := range templatesMap {
		templates = append(templates, template)
	}

	slices.SortFunc(templates, func(a, b WidgetTemplate) int {
		return strings.Compare(string(a.Code()), string(b.Code()))
	})

	return templates
}

func (r *runtimeWidgetTemplateRegistry) fullWidgetCode(code WidgetCode) (WidgetCode, error) {
	if code == "" {
		return "", errors.New("widget code is empty")
	}

	if strings.Contains(string(code), ".") {
		return code, nil
	}

	if r.moduleCode == "" {
		return "", errors.New("module code is empty")
	}

	return WidgetCode(r.moduleCode + "." + string(code)), nil
}
