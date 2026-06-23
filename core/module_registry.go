package core

import "fmt"

type ModuleRegistry struct {
	Widgets []Widget
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

	return nil
}
