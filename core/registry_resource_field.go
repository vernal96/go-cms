package core

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type runtimeResourceFieldRegistry struct {
	resourceTypes     map[ResourceType]ResourceTypeDefinition
	resourceTemplates map[ResourceType]map[ResourceTemplateCode]ResourceTemplateDefinition
	resourceFields    map[ResourceType]map[ResourceTemplateCode]map[ResourceFieldCode]ResourceFieldDefinition
}

func (r *runtimeResourceFieldRegistry) Register(field ResourceFieldDefinition) error {
	if field == nil {
		return errors.New("resource field is nil")
	}

	code := field.Code()
	if code == "" {
		return errors.New("resource field code is empty")
	}

	if field.Name() == "" {
		return errors.New("resource field name is empty")
	}

	if field.Field() == nil {
		return errors.New("resource field type is nil")
	}

	resourceType := field.ResourceType()
	if resourceType == "" {
		return errors.New("resource field resource type is empty")
	}

	template := field.ResourceTemplate()
	if template == "" {
		return errors.New("resource field resource template is empty")
	}

	if _, exists := r.resourceTypes[resourceType]; !exists {
		return fmt.Errorf("resource type %q is not registered", resourceType)
	}

	templates, exists := r.resourceTemplates[resourceType]
	if !exists {
		return fmt.Errorf(
			"resource template %q for resource type %q is not registered",
			template,
			resourceType,
		)
	}
	if _, exists := templates[template]; !exists {
		return fmt.Errorf(
			"resource template %q for resource type %q is not registered",
			template,
			resourceType,
		)
	}

	if _, exists := r.resourceFields[resourceType]; !exists {
		r.resourceFields[resourceType] = make(
			map[ResourceTemplateCode]map[ResourceFieldCode]ResourceFieldDefinition,
		)
	}
	if _, exists := r.resourceFields[resourceType][template]; !exists {
		r.resourceFields[resourceType][template] = make(
			map[ResourceFieldCode]ResourceFieldDefinition,
		)
	}

	if _, exists := r.resourceFields[resourceType][template][code]; exists {
		return fmt.Errorf(
			"resource field %q for resource type %q and template %q is already registered",
			code,
			resourceType,
			template,
		)
	}

	r.resourceFields[resourceType][template][code] = field

	return nil
}

func (r *runtimeResourceFieldRegistry) Get(
	resourceType ResourceType,
	template ResourceTemplateCode,
	code ResourceFieldCode,
) (ResourceFieldDefinition, bool) {
	templates, exists := r.resourceFields[resourceType]
	if !exists {
		return nil, false
	}

	fields, exists := templates[template]
	if !exists {
		return nil, false
	}

	field, exists := fields[code]

	return field, exists
}

func (r *runtimeResourceFieldRegistry) AllForTemplate(
	resourceType ResourceType,
	template ResourceTemplateCode,
) []ResourceFieldDefinition {
	templates, exists := r.resourceFields[resourceType]
	if !exists {
		return nil
	}

	fieldsMap, exists := templates[template]
	if !exists {
		return nil
	}

	fields := make([]ResourceFieldDefinition, 0, len(fieldsMap))
	for _, field := range fieldsMap {
		fields = append(fields, field)
	}

	slices.SortFunc(fields, func(a, b ResourceFieldDefinition) int {
		return strings.Compare(string(a.Code()), string(b.Code()))
	})

	return fields
}

var _ ResourceFieldRegistry = (*runtimeResourceFieldRegistry)(nil)
