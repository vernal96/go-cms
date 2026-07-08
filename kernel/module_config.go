package kernel

import (
	"fmt"
	"reflect"
)

func ModuleConfigFrom[T any](moduleContext ModuleContext) (T, error) {
	var moduleConfig T

	rawModuleConfig := moduleContext.ModuleConfig()
	if rawModuleConfig == nil {
		return moduleConfig, nil
	}

	if err := mergeModuleConfig(&moduleConfig, rawModuleConfig); err != nil {
		return moduleConfig, err
	}

	return moduleConfig, nil
}

func mergeModuleConfig(target any, source any) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Pointer || targetValue.IsNil() {
		return fmt.Errorf("module config target must be a non-nil pointer")
	}

	sourceValue := reflect.ValueOf(source)
	if sourceValue.Kind() == reflect.Pointer {
		if sourceValue.IsNil() {
			return fmt.Errorf("module config source is nil")
		}

		sourceValue = sourceValue.Elem()
	}

	targetElement := targetValue.Elem()

	if sourceValue.Type() != targetElement.Type() {
		return fmt.Errorf(
			"invalid module config type %s, expected %s",
			sourceValue.Type(),
			targetElement.Type(),
		)
	}

	mergeModuleConfigValues(targetElement, sourceValue)

	return nil
}

func mergeModuleConfigValues(target reflect.Value, source reflect.Value) {
	for i := 0; i < source.NumField(); i++ {
		sourceField := source.Field(i)
		targetField := target.Field(i)

		if !targetField.CanSet() {
			continue
		}

		if sourceField.Kind() == reflect.Struct && targetField.Kind() == reflect.Struct {
			mergeModuleConfigValues(targetField, sourceField)
			continue
		}

		if sourceField.IsZero() {
			continue
		}

		targetField.Set(sourceField)
	}
}
