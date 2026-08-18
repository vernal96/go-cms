package app

import (
	"fmt"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/permission"
)

func validateApplicationProfile(profile kernel.Profile) error {
	moduleCodes := make(map[kernel.ModuleCode]struct{}, len(profile.Modules))
	for moduleIndex, profileModule := range profile.Modules {
		if profileModule.Module == nil {
			return fmt.Errorf(
				"profile %q module at index %d is nil",
				profile.Code,
				moduleIndex,
			)
		}
		moduleCode := profileModule.Module.Code()
		if moduleCode == "" {
			return fmt.Errorf(
				"profile %q module at index %d has empty code",
				profile.Code,
				moduleIndex,
			)
		}
		if _, exists := moduleCodes[moduleCode]; exists {
			return fmt.Errorf(
				"profile %q contains duplicate module %q",
				profile.Code,
				moduleCode,
			)
		}
		moduleCodes[moduleCode] = struct{}{}
	}
	for _, required := range requiredModuleCodes {
		if _, exists := moduleCodes[required]; !exists {
			return fmt.Errorf(
				"profile %q does not contain required module %q",
				profile.Code,
				required,
			)
		}
	}
	if profile.Modules[0].Module.Code() != core.ModuleCode {
		return fmt.Errorf(
			"profile %q must declare module %q first",
			profile.Code,
			core.ModuleCode,
		)
	}
	return nil
}

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
