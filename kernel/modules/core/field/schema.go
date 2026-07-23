package field

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/go-playground/validator/v10"
)

type compiledField struct {
	definition Definition
	valueType  ValueType
	required   bool
	rules      string
}

type Schema struct {
	definitions []Definition
	fields      map[string]compiledField
	validator   *validator.Validate
}

func Compile(
	definitions []Definition,
	resolver TypeResolver,
) (*Schema, error) {
	if resolver == nil {
		return nil, errors.New("field type resolver is nil")
	}

	definitions = CloneDefinitions(definitions)
	schema := &Schema{
		definitions: definitions,
		fields:      make(map[string]compiledField, len(definitions)),
		validator:   validator.New(),
	}

	for index, definition := range definitions {
		if definition.Key == "" ||
			strings.TrimSpace(definition.Key) != definition.Key {
			return nil, fmt.Errorf(
				"field at index %d has invalid key %q",
				index,
				definition.Key,
			)
		}
		if definition.Label == "" ||
			strings.TrimSpace(definition.Label) != definition.Label {
			return nil, fmt.Errorf(
				"field %q has invalid label %q",
				definition.Key,
				definition.Label,
			)
		}
		if definition.Type == "" {
			return nil, fmt.Errorf(
				"field %q has empty type",
				definition.Key,
			)
		}
		if _, exists := schema.fields[definition.Key]; exists {
			return nil, fmt.Errorf(
				"duplicate field key %q",
				definition.Key,
			)
		}

		fieldType, exists := resolver.FieldType(definition.Type)
		if !exists {
			return nil, fmt.Errorf(
				"field %q references unknown type %q",
				definition.Key,
				definition.Type,
			)
		}
		if nilInterface(fieldType) {
			return nil, fmt.Errorf(
				"field %q type %q is nil",
				definition.Key,
				definition.Type,
			)
		}

		valueType, err := fieldType.Compile(definition.Options)
		if err != nil {
			return nil, fmt.Errorf(
				"compile field %q type %q: %w",
				definition.Key,
				definition.Type,
				err,
			)
		}
		if nilInterface(valueType) {
			return nil, fmt.Errorf(
				"field type %q returned nil value type",
				definition.Type,
			)
		}

		rules, err := compileRules(valueType.Rules(), definition.Rules)
		if err != nil {
			return nil, fmt.Errorf(
				"compile field %q rules: %w",
				definition.Key,
				err,
			)
		}
		if rules != "" {
			err := safeValidate(
				schema.validator,
				definition.Key,
				valueType.Example(),
				rules,
			)
			var validationErrors validator.ValidationErrors
			if err != nil && !errors.As(err, &validationErrors) {
				return nil, fmt.Errorf(
					"compile field %q rules: %w",
					definition.Key,
					err,
				)
			}
		}

		schema.fields[definition.Key] = compiledField{
			definition: definition,
			valueType:  valueType,
			required: definition.Required != nil &&
				*definition.Required,
			rules: rules,
		}
	}

	return schema, nil
}

func (s *Schema) Definitions() []Definition {
	if s == nil {
		return nil
	}

	return CloneDefinitions(s.definitions)
}

func (s *Schema) Validate(
	values map[string]any,
) (map[string]any, error) {
	if s == nil {
		return nil, errors.New("field schema is nil")
	}

	result := make(map[string]any, len(values))
	validationErrors := make(ValidationErrors, 0)

	unknownKeys := make([]string, 0)
	for key := range values {
		if _, exists := s.fields[key]; !exists {
			unknownKeys = append(unknownKeys, key)
		}
	}
	sort.Strings(unknownKeys)
	for _, key := range unknownKeys {
		validationErrors = append(validationErrors, ValidationError{
			Key:  key,
			Rule: "defined",
		})
	}

	for _, definition := range s.definitions {
		compiled := s.fields[definition.Key]
		value, exists := values[definition.Key]

		if !exists || inputEmpty(value) {
			if compiled.required {
				validationErrors = append(
					validationErrors,
					ValidationError{
						Key:  definition.Key,
						Rule: "required",
					},
				)
			}
			continue
		}

		normalized, err := compiled.valueType.Normalize(value)
		if err != nil {
			validationErrors = append(
				validationErrors,
				ValidationError{
					Key:  definition.Key,
					Rule: "type",
				},
			)
			continue
		}

		if compiled.valueType.Empty(normalized) {
			if compiled.required {
				validationErrors = append(
					validationErrors,
					ValidationError{
						Key:  definition.Key,
						Rule: "required",
					},
				)
			}
			continue
		}

		if err := compiled.valueType.Validate(normalized); err != nil {
			if ruleError, exists := ruleErrorFrom(err); exists {
				validationErrors = append(
					validationErrors,
					ValidationError{
						Key:   definition.Key,
						Rule:  ruleError.Rule,
						Param: ruleError.Param,
					},
				)
			} else {
				validationErrors = append(
					validationErrors,
					ValidationError{
						Key:  definition.Key,
						Rule: "value",
					},
				)
			}
			continue
		}

		if compiled.rules != "" {
			err := safeValidate(
				s.validator,
				definition.Key,
				normalized,
				compiled.rules,
			)
			if err != nil {
				var fieldErrors validator.ValidationErrors
				if errors.As(err, &fieldErrors) {
					for _, fieldError := range fieldErrors {
						validationErrors = append(
							validationErrors,
							ValidationError{
								Key:   definition.Key,
								Rule:  fieldError.Tag(),
								Param: fieldError.Param(),
							},
						)
					}
				} else {
					validationErrors = append(
						validationErrors,
						ValidationError{
							Key:  definition.Key,
							Rule: "validation",
						},
					)
				}
				continue
			}
		}

		result[definition.Key] = normalized
	}

	if len(validationErrors) > 0 {
		return nil, validationErrors
	}

	return result, nil
}

func ruleErrorFrom(err error) (RuleError, bool) {
	var value RuleError
	if errors.As(err, &value) {
		return value, true
	}

	var pointer *RuleError
	if errors.As(err, &pointer) && pointer != nil {
		return *pointer, true
	}

	return RuleError{}, false
}

func compileRules(defaults, configured []string) (string, error) {
	rules := make([]string, 0, len(defaults)+len(configured))
	seen := make(map[string]struct{}, len(defaults)+len(configured))

	for _, source := range [][]string{defaults, configured} {
		for _, rule := range source {
			rule = strings.TrimSpace(rule)
			if rule == "" {
				return "", errors.New("validation rule is empty")
			}
			if strings.Contains(rule, ",") {
				return "", fmt.Errorf(
					"validation rule %q must be a single tag",
					rule,
				)
			}

			name := rule
			if separator := strings.IndexByte(name, '='); separator >= 0 {
				name = name[:separator]
			}
			if name == "required" || name == "omitempty" {
				return "", fmt.Errorf(
					"validation rule %q is managed by Required",
					name,
				)
			}
			if _, exists := seen[rule]; exists {
				continue
			}

			seen[rule] = struct{}{}
			rules = append(rules, rule)
		}
	}

	return strings.Join(rules, ","), nil
}

func safeValidate(
	validate *validator.Validate,
	key string,
	value any,
	rules string,
) (resultErr error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			resultErr = fmt.Errorf(
				"invalid validation rules %q: %v",
				rules,
				recovered,
			)
		}
	}()

	return validate.VarWithKey(key, value, rules)
}
