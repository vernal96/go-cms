package kernel

import "fmt"

func ModuleConfigFrom[T any](moduleContext ModuleContext) (T, error) {
	var zero T

	rawModuleConfig := moduleContext.ModuleConfig()
	if rawModuleConfig == nil {
		return zero, nil
	}

	if moduleConfig, ok := rawModuleConfig.(T); ok {
		return moduleConfig, nil
	}

	if moduleConfig, ok := rawModuleConfig.(*T); ok {
		if moduleConfig == nil {
			return zero, fmt.Errorf("module config is nil")
		}

		return *moduleConfig, nil
	}

	return zero, fmt.Errorf(
		"invalid module config type %T, expected %T",
		rawModuleConfig,
		zero,
	)
}
