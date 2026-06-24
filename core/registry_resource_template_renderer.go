package core

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type runtimeResourceTemplateRendererRegistry struct {
	resourceTypes             map[ResourceType]ResourceTypeDefinition
	resourceTemplates         map[ResourceType]map[ResourceTemplateCode]ResourceTemplateDefinition
	resourceTemplateRenderers map[ResourceType]map[ResourceTemplateCode]ResourceTemplateRenderer
}

func (r *runtimeResourceTemplateRendererRegistry) Register(
	renderer ResourceTemplateRenderer,
) error {
	if renderer == nil {
		return errors.New("resource template renderer is nil")
	}

	resourceType := renderer.ResourceType()
	if resourceType == "" {
		return errors.New("resource template renderer resource type is empty")
	}

	template := renderer.ResourceTemplate()
	if template == "" {
		return errors.New("resource template renderer template is empty")
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

	if _, exists := r.resourceTemplateRenderers[resourceType]; !exists {
		r.resourceTemplateRenderers[resourceType] = make(
			map[ResourceTemplateCode]ResourceTemplateRenderer,
		)
	}

	if _, exists := r.resourceTemplateRenderers[resourceType][template]; exists {
		return fmt.Errorf(
			"resource template renderer for resource type %q and template %q is already registered",
			resourceType,
			template,
		)
	}

	r.resourceTemplateRenderers[resourceType][template] = renderer

	return nil
}

func (r *runtimeResourceTemplateRendererRegistry) Get(
	resourceType ResourceType,
	template ResourceTemplateCode,
) (ResourceTemplateRenderer, bool) {
	renderers, exists := r.resourceTemplateRenderers[resourceType]
	if !exists {
		return nil, false
	}

	renderer, exists := renderers[template]

	return renderer, exists
}

func (r *runtimeResourceTemplateRendererRegistry) AllForResourceType(
	resourceType ResourceType,
) []ResourceTemplateRenderer {
	renderersMap, exists := r.resourceTemplateRenderers[resourceType]
	if !exists {
		return nil
	}

	renderers := make([]ResourceTemplateRenderer, 0, len(renderersMap))
	for _, renderer := range renderersMap {
		renderers = append(renderers, renderer)
	}

	slices.SortFunc(renderers, func(a, b ResourceTemplateRenderer) int {
		return strings.Compare(
			string(a.ResourceTemplate()),
			string(b.ResourceTemplate()),
		)
	})

	return renderers
}

var _ ResourceTemplateRendererRegistry = (*runtimeResourceTemplateRendererRegistry)(nil)
