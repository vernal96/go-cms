package core

import "fmt"

type ModuleRegistry struct {
	ResourceTypes     []ResourceTypeDefinition
	ResourceTemplates []ResourceTemplateDefinition
	ResourceFields    []ResourceFieldDefinition
	Widgets           []Widget
	Controllers       []Controller
}

func RegisterModule(registry Registry, moduleRegistry ModuleRegistry) error {
	if registry == nil {
		return fmt.Errorf("registry is nil")
	}

	for _, resourceType := range moduleRegistry.ResourceTypes {
		if err := registry.ResourceTypes().Register(resourceType); err != nil {
			return fmt.Errorf("register resource type: %w", err)
		}
	}

	for _, template := range moduleRegistry.ResourceTemplates {
		if err := registry.ResourceTemplates().Register(template); err != nil {
			return fmt.Errorf("register resource template: %w", err)
		}
	}

	for _, field := range moduleRegistry.ResourceFields {
		if err := registry.ResourceFields().Register(field); err != nil {
			return fmt.Errorf("register resource field: %w", err)
		}
	}

	for _, widget := range moduleRegistry.Widgets {
		if err := registry.Widgets().Register(widget); err != nil {
			return fmt.Errorf("register widget %q: %w", widget.Code(), err)
		}
	}

	for _, controller := range moduleRegistry.Controllers {
		if err := registry.Controllers().Register(controller); err != nil {
			return fmt.Errorf("register controller: %w", err)
		}
	}

	return nil
}
