package core

import "fmt"

type ModuleRegistry struct {
	Widgets     []Widget
	Controllers []Controller
}

func RegisterModule(registry Registry, moduleRegistry ModuleRegistry) error {
	if registry == nil {
		return fmt.Errorf("registry is nil")
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
