package management

import (
	"fmt"

	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
)

const e164Pattern = `^\+[1-9][0-9]{1,14}$`

type FieldChoice struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type FieldOptions struct {
	Step      *float64          `json:"step,omitempty"`
	Choices   []FieldChoice     `json:"choices,omitempty"`
	Multiple  *bool             `json:"multiple,omitempty"`
	Pattern   *string           `json:"pattern,omitempty"`
	Storages  []filesystem.Code `json:"storages,omitempty"`
	MIMETypes []string          `json:"mime_types,omitempty"`
}

type FieldVisibleWhen struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

type FieldDefinition struct {
	Key         string            `json:"key"`
	Type        field.TypeCode    `json:"type"`
	Label       string            `json:"label"`
	Required    bool              `json:"required"`
	Rules       []string          `json:"rules"`
	Options     *FieldOptions     `json:"options,omitempty"`
	Editor      field.EditorCode  `json:"editor,omitempty"`
	VisibleWhen *FieldVisibleWhen `json:"visible_when,omitempty"`
}

type FieldValidationError struct {
	Key   string `json:"key"`
	Rule  string `json:"rule"`
	Param string `json:"param"`
}

type ValidationError struct {
	Message string
	Fields  []FieldValidationError
}

func (e ValidationError) Error() string {
	return e.Message
}

func (e ValidationError) Unwrap() error {
	return ErrValidation
}

func fieldDefinitions(source []field.Definition) ([]FieldDefinition, error) {
	result := make([]FieldDefinition, len(source))
	for index, definition := range source {
		converted, err := fieldDefinition(definition)
		if err != nil {
			return nil, err
		}
		result[index] = converted
	}
	return result, nil
}

func fieldDefinition(definition field.Definition) (FieldDefinition, error) {
	result := FieldDefinition{
		Key:      definition.Key,
		Type:     definition.Type,
		Label:    definition.Label,
		Required: definition.Required != nil && *definition.Required,
		Rules:    append([]string(nil), definition.Rules...),
		Editor:   definition.Editor,
	}
	if definition.VisibleWhen != nil {
		result.VisibleWhen = &FieldVisibleWhen{Field: definition.VisibleWhen.Field, Value: definition.VisibleWhen.Value}
	}

	if result.Rules == nil {
		result.Rules = []string{}
	}

	switch definition.Type {
	case field.TypeString, field.TypeCheckbox, field.TypeTextarea, field.TypeEmail, field.TypeJSON:
		if definition.Options != nil {
			return FieldDefinition{}, fmt.Errorf(
				"CMS field %q type %q has unsupported options %T",
				definition.Key, definition.Type, definition.Options,
			)
		}

	case field.TypeInteger:
		options, err := integerFieldOptions(definition.Options)
		if err != nil {
			return FieldDefinition{}, fieldOptionsError(definition, err)
		}
		result.Options = &FieldOptions{}
		if options.Step != nil {
			step := float64(*options.Step)
			result.Options.Step = &step
		}

	case field.TypeFloat:
		options, err := floatFieldOptions(definition.Options)
		if err != nil {
			return FieldDefinition{}, fieldOptionsError(definition, err)
		}
		result.Options = &FieldOptions{Step: cloneFloat(options.Step)}

	case field.TypeRadio:
		options, err := radioFieldOptions(definition.Options)
		if err != nil {
			return FieldDefinition{}, fieldOptionsError(definition, err)
		}
		result.Options = &FieldOptions{Choices: fieldChoices(options.Choices)}

	case field.TypeSelect:
		options, err := selectFieldOptions(definition.Options)
		if err != nil {
			return FieldDefinition{}, fieldOptionsError(definition, err)
		}
		multiple := options.Multiple
		result.Options = &FieldOptions{
			Choices:  fieldChoices(options.Choices),
			Multiple: &multiple,
		}

	case field.TypePhone:
		options, err := phoneFieldOptions(definition.Options)
		if err != nil {
			return FieldDefinition{}, fieldOptionsError(definition, err)
		}
		pattern := options.Pattern
		if pattern == "" {
			pattern = e164Pattern
		}
		result.Options = &FieldOptions{Pattern: &pattern}

	case field.TypeFile:
		options, err := field.FileOptionsValue(definition.Options)
		if err != nil {
			return FieldDefinition{}, fieldOptionsError(definition, err)
		}
		result.Options = &FieldOptions{
			Storages:  append([]filesystem.Code(nil), options.Storages...),
			MIMETypes: append([]string(nil), options.MIMETypes...),
		}

	default:
		return FieldDefinition{}, fmt.Errorf(
			"CMS field %q has unsupported type %q",
			definition.Key, definition.Type,
		)
	}

	return result, nil
}

func integerFieldOptions(value any) (field.IntegerOptions, error) {
	switch options := value.(type) {
	case nil:
		return field.IntegerOptions{}, nil
	case field.IntegerOptions:
		return options, nil
	case *field.IntegerOptions:
		if options != nil {
			return *options, nil
		}
	}
	return field.IntegerOptions{}, fmt.Errorf("got %T", value)
}

func floatFieldOptions(value any) (field.FloatOptions, error) {
	switch options := value.(type) {
	case nil:
		return field.FloatOptions{}, nil
	case field.FloatOptions:
		return options, nil
	case *field.FloatOptions:
		if options != nil {
			return *options, nil
		}
	}
	return field.FloatOptions{}, fmt.Errorf("got %T", value)
}

func radioFieldOptions(value any) (field.RadioOptions, error) {
	switch options := value.(type) {
	case field.RadioOptions:
		return options, nil
	case *field.RadioOptions:
		if options != nil {
			return *options, nil
		}
	}
	return field.RadioOptions{}, fmt.Errorf("got %T", value)
}

func selectFieldOptions(value any) (field.SelectOptions, error) {
	switch options := value.(type) {
	case field.SelectOptions:
		return options, nil
	case *field.SelectOptions:
		if options != nil {
			return *options, nil
		}
	}
	return field.SelectOptions{}, fmt.Errorf("got %T", value)
}

func phoneFieldOptions(value any) (field.PhoneOptions, error) {
	switch options := value.(type) {
	case nil:
		return field.PhoneOptions{}, nil
	case field.PhoneOptions:
		return options, nil
	case *field.PhoneOptions:
		if options != nil {
			return *options, nil
		}
	}
	return field.PhoneOptions{}, fmt.Errorf("got %T", value)
}

func fieldOptionsError(definition field.Definition, err error) error {
	return fmt.Errorf(
		"CMS field %q type %q has invalid options: %w",
		definition.Key, definition.Type, err,
	)
}

func fieldChoices(source []field.Choice) []FieldChoice {
	result := make([]FieldChoice, len(source))
	for index, choice := range source {
		result[index] = FieldChoice{Value: choice.Value, Label: choice.Label}
	}
	return result
}

func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}
