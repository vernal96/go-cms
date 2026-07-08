package kernel

import (
	"fmt"
	"reflect"
)

func ConfigFrom[T any](moduleContext ModuleContext) (T, error) {
	var config T

	if err := applyConfigDefaults(&config); err != nil {
		return config, err
	}

	rawConfig := moduleContext.Config()
	if rawConfig == nil {
		return config, nil
	}

	if err := mergeConfig(&config, rawConfig); err != nil {
		return config, err
	}

	return config, nil
}

func applyConfigDefaults(target any) error {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("config target must be a non-nil pointer")
	}

	return applyDefaultsToValue(value.Elem())
}

func applyDefaultsToValue(value reflect.Value) error {
	if value.Kind() != reflect.Struct {
		return nil
	}

	valueType := value.Type()

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		structField := valueType.Field(i)

		if !field.CanSet() {
			continue
		}

		if field.Kind() == reflect.Struct {
			if err := applyDefaultsToValue(field); err != nil {
				return err
			}
		}

		defaultValue := structField.Tag.Get("default")
		if defaultValue == "" {
			continue
		}

		if !field.IsZero() {
			continue
		}

		switch field.Kind() {
		case reflect.String:
			field.SetString(defaultValue)

		default:
			return fmt.Errorf("unsupported default field type %s for field %s",
				field.Kind(),
				structField.Name)
		}
	}

	return nil
}

func mergeConfig(target any, source any) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Pointer || targetValue.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	sourceValue := reflect.ValueOf(source)
	if sourceValue.Kind() != reflect.Pointer {
		if sourceValue.IsNil() {
			return fmt.Errorf("config source is nil")
		}

		sourceValue = sourceValue.Elem()
	}

	targetElem := targetValue.Elem()

	if sourceValue.Type() != targetElem.Type() {
		return fmt.Errorf(
			"invalid config type %s, expected %s",
			sourceValue.Type(),
			targetElem.Type(),
		)
	}

	mergeStructValues(targetElem, sourceValue)

	return nil
}

func mergeStructValues(target reflect.Value, source reflect.Value) {
	for i := 0; i < source.NumField(); i++ {
		sourceField := source.Field(i)
		targetField := target.Field(i)

		if !targetField.CanSet() {
			continue
		}

		if sourceField.Kind() == reflect.Struct && targetField.Kind() == reflect.Struct {
			mergeStructValues(targetField, sourceField)
			continue
		}

		if sourceField.IsZero() {
			continue
		}

		targetField.Set(sourceField)
	}
}
