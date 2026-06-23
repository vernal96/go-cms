package core

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

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

var _ WidgetTemplateRegistry = (*runtimeWidgetTemplateRegistry)(nil)
