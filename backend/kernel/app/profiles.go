package app

import (
	"fmt"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/permission"
)

func buildPermissionCatalog(
	profiles []kernel.Profile,
) (*permission.Catalog, error) {
	seenModules := make(map[kernel.ModuleCode]struct{})
	definitions := make([]permission.Definition, 0)

	for _, profile := range profiles {
		for _, profileModule := range profile.Modules {
			if profileModule.Module == nil {
				continue
			}
			moduleCode := profileModule.Module.Code()
			if _, exists := seenModules[moduleCode]; exists {
				continue
			}
			seenModules[moduleCode] = struct{}{}

			provider, exists := profileModule.Module.(kernel.RegistryProvider)
			if !exists {
				continue
			}
			moduleDefinitions, err := permission.Definitions(
				string(moduleCode),
				provider.Registry().PermissionEntities,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"build permissions for module %q: %w",
					moduleCode,
					err,
				)
			}
			definitions = append(definitions, moduleDefinitions...)
		}
	}

	return permission.NewCatalog(definitions)
}

type profileResolver map[kernel.ProfileCode]*kernel.ProfileBlueprint

func (r profileResolver) ProfileBlueprint(
	code kernel.ProfileCode,
) (*kernel.ProfileBlueprint, bool) {
	blueprint, exists := r[code]
	return blueprint, exists
}
