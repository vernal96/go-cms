package core

import "fmt"

func RegisterModuleEntities(registry Registry, entities ModuleEntities) error {
	if registry == nil {
		return fmt.Errorf("registry is nil")
	}

	for _, widget := range entities.Widgets {
		if err := registry.Widgets().Register(widget); err != nil {
			return fmt.Errorf("register widget %q: %w", widget.Code(), err)
		}
	}

	return nil
}
