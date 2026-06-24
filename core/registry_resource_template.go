package core

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type runtimeResourceTemplateRegistry struct {
	resourceTypes     map[ResourceType]ResourceTypeDefinition
	resourceTemplates map[ResourceType]map[ResourceTemplateCode]ResourceTemplateDefinition
}

func (r *runtimeResourceTemplateRegistry) Register(template ResourceTemplateDefinition) error {
	if template == nil {
		return errors.New("resource template is nil")
	}

	code := template.Code()
	if code == "" {
		return errors.New("resource template code is empty")
	}

	resourceType := template.ResourceType()
	if resourceType == "" {
		return errors.New("resource template resource type is empty")
	}

	if _, exists := r.resourceTypes[resourceType]; !exists {
		return fmt.Errorf("resource type %q is not registered", resourceType)
	}

	if _, exists := r.resourceTemplates[resourceType]; !exists {
		r.resourceTemplates[resourceType] = make(map[ResourceTemplateCode]ResourceTemplateDefinition)
	}

	if _, exists := r.resourceTemplates[resourceType][code]; exists {
		return fmt.Errorf(
			"resource template %q for resource type %q is already registered",
			code,
			resourceType,
		)
	}

	r.resourceTemplates[resourceType][code] = template

	return nil
}

func (r *runtimeResourceTemplateRegistry) Get(
	resourceType ResourceType,
	code ResourceTemplateCode,
) (ResourceTemplateDefinition, bool) {
	templates, exists := r.resourceTemplates[resourceType]
	if !exists {
		return nil, false
	}

	template, exists := templates[code]

	return template, exists
}

func (r *runtimeResourceTemplateRegistry) AllForResourceType(
	resourceType ResourceType,
) []ResourceTemplateDefinition {
	templatesMap, exists := r.resourceTemplates[resourceType]
	if !exists {
		return nil
	}

	templates := make([]ResourceTemplateDefinition, 0, len(templatesMap))
	for _, template := range templatesMap {
		templates = append(templates, template)
	}

	slices.SortFunc(templates, func(a, b ResourceTemplateDefinition) int {
		return strings.Compare(string(a.Code()), string(b.Code()))
	})

	return templates
}

var _ ResourceTemplateRegistry = (*runtimeResourceTemplateRegistry)(nil)
