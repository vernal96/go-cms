package kernel

import (
	"fmt"
	"reflect"
	"strings"
)

const moduleConfigTagName = "cms"

func ModuleConfigFrom[T any](moduleContext ModuleContext) (T, error) {
	var moduleConfig T

	rawModuleConfig := moduleContext.ModuleConfig()
	if rawModuleConfig != nil {
		if err := mergeModuleConfig(&moduleConfig, rawModuleConfig); err != nil {
			return moduleConfig, err
		}
	}

	if err := resolveModuleConfig(
		reflect.ValueOf(&moduleConfig).Elem(),
		moduleContext.AdapterDefaults(),
		moduleContext,
	); err != nil {
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

func resolveModuleConfig(
	value reflect.Value,
	adapterDefaults AdapterDefaults,
	moduleContext ModuleContext,
) error {
	if value.Kind() != reflect.Struct {
		return nil
	}

	currentAdapterDefaults := resolveStructAdapterDefaults(value, adapterDefaults)
	valueType := value.Type()

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		structField := valueType.Field(i)

		if !field.CanSet() {
			continue
		}

		if isAdapterDefaultsField(field) {
			continue
		}

		if field.Kind() == reflect.Struct {
			if err := resolveModuleConfig(field, currentAdapterDefaults, moduleContext); err != nil {
				return err
			}
		}

		tag := parseModuleConfigTag(structField.Tag.Get(moduleConfigTagName))

		if tag.adapter {
			if err := resolveAdapterConfigField(field, structField, tag, currentAdapterDefaults, moduleContext); err != nil {
				return err
			}
		}

		if tag.required && field.IsZero() {
			return fmt.Errorf("required module config field %s is empty", structField.Name)
		}
	}

	return nil
}

func resolveStructAdapterDefaults(value reflect.Value, parent AdapterDefaults) AdapterDefaults {
	result := parent

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		if !isAdapterDefaultsField(field) || !field.CanInterface() {
			continue
		}

		result = ResolveAdapterDefaults(result, field.Interface().(AdapterDefaults))
		break
	}

	return result
}

func resolveAdapterConfigField(
	field reflect.Value,
	structField reflect.StructField,
	tag moduleConfigTag,
	adapterDefaults AdapterDefaults,
	moduleContext ModuleContext,
) error {
	if field.Kind() != reflect.String {
		return fmt.Errorf("adapter module config field %s must be string-based", structField.Name)
	}

	if field.IsZero() {
		adapterCode, err := adapterCodeFromDefaults(adapterDefaults, tag.defaultName)
		if err != nil {
			return err
		}

		field.SetString(string(adapterCode))
	}

	if tag.contract == "" {
		return fmt.Errorf("adapter module config field %s has empty contract", structField.Name)
	}

	adapterCode := AdapterCode(field.String())
	if _, exists := moduleContext.App().Adapters().Get(tag.contract, adapterCode); !exists {
		return fmt.Errorf("adapter %q for contract %q is not registered", adapterCode, tag.contract)
	}

	return nil
}

func adapterCodeFromDefaults(defaults AdapterDefaults, defaultName string) (AdapterCode, error) {
	if defaultName == "" {
		return AdapterDefault, nil
	}

	switch defaultName {
	case "repository":
		return defaults.RepositoryAdapter, nil

	default:
		return "", fmt.Errorf("unknown adapter default %q", defaultName)
	}
}

func isAdapterDefaultsField(field reflect.Value) bool {
	return field.Type() == reflect.TypeOf(AdapterDefaults{})
}

type moduleConfigTag struct {
	adapter     bool
	required    bool
	contract    AdapterContractCode
	defaultName string
}

func parseModuleConfigTag(rawTag string) moduleConfigTag {
	result := moduleConfigTag{}

	if rawTag == "" {
		return result
	}

	parts := strings.Split(rawTag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		switch part {
		case "adapter":
			result.adapter = true
		case "required":
			result.required = true
		default:
			key, value, found := strings.Cut(part, "=")
			if !found {
				continue
			}

			switch strings.TrimSpace(key) {
			case "contract":
				result.contract = AdapterContractCode(strings.TrimSpace(value))
			case "default":
				result.defaultName = strings.TrimSpace(value)
			}
		}
	}

	return result
}
