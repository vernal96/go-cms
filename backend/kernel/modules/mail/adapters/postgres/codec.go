package postgres

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
)

type variableJSON struct {
	Key      string         `json:"key"`
	Type     field.TypeCode `json:"type"`
	Label    string         `json:"label"`
	Required bool           `json:"required"`
	Rules    []string       `json:"rules"`
	Options  *optionsJSON   `json:"options,omitempty"`
}

type optionsJSON struct {
	Step      *float64          `json:"step,omitempty"`
	Choices   []choiceJSON      `json:"choices,omitempty"`
	Multiple  *bool             `json:"multiple,omitempty"`
	Pattern   *string           `json:"pattern,omitempty"`
	Storages  []filesystem.Code `json:"storages,omitempty"`
	MIMETypes []string          `json:"mime_types,omitempty"`
}

type choiceJSON struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func encodeVariables(definitions []field.Definition) ([]byte, error) {
	items := make([]variableJSON, len(definitions))
	for index, definition := range definitions {
		item := variableJSON{Key: definition.Key, Type: definition.Type, Label: definition.Label, Required: definition.Required != nil && *definition.Required, Rules: append([]string(nil), definition.Rules...)}
		options, err := encodeOptions(definition)
		if err != nil {
			return nil, err
		}
		item.Options = options
		items[index] = item
	}
	return json.Marshal(items)
}

func decodeVariables(raw []byte) ([]field.Definition, error) {
	var items []variableJSON
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, err
	}
	result := make([]field.Definition, len(items))
	for index, item := range items {
		required := item.Required
		options, err := decodeOptions(item.Type, item.Options)
		if err != nil {
			return nil, fmt.Errorf("decode variable %q options: %w", item.Key, err)
		}
		result[index] = field.Definition{Key: item.Key, Type: item.Type, Label: item.Label, Required: &required, Rules: append([]string(nil), item.Rules...), Options: options}
	}
	return result, nil
}

func encodeOptions(definition field.Definition) (*optionsJSON, error) {
	switch definition.Type {
	case field.TypeString, field.TypeTextarea, field.TypeEmail, field.TypeCheckbox:
		if definition.Options != nil {
			return nil, fmt.Errorf("variable %q has unsupported options", definition.Key)
		}
		return nil, nil
	case field.TypeInteger:
		options, ok := definition.Options.(field.IntegerOptions)
		if definition.Options == nil {
			return &optionsJSON{}, nil
		}
		if !ok {
			if pointer, pointerOK := definition.Options.(*field.IntegerOptions); pointerOK && pointer != nil {
				options = *pointer
			} else {
				return nil, fmt.Errorf("variable %q integer options have type %T", definition.Key, definition.Options)
			}
		}
		result := &optionsJSON{}
		if options.Step != nil {
			step := float64(*options.Step)
			result.Step = &step
		}
		return result, nil
	case field.TypeFloat:
		options, ok := definition.Options.(field.FloatOptions)
		if definition.Options == nil {
			return &optionsJSON{}, nil
		}
		if !ok {
			if pointer, pointerOK := definition.Options.(*field.FloatOptions); pointerOK && pointer != nil {
				options = *pointer
			} else {
				return nil, fmt.Errorf("variable %q float options have type %T", definition.Key, definition.Options)
			}
		}
		return &optionsJSON{Step: options.Step}, nil
	case field.TypeRadio:
		options, ok := definition.Options.(field.RadioOptions)
		if !ok {
			return nil, fmt.Errorf("variable %q radio options have type %T", definition.Key, definition.Options)
		}
		return &optionsJSON{Choices: encodeChoices(options.Choices)}, nil
	case field.TypeSelect:
		options, ok := definition.Options.(field.SelectOptions)
		if !ok {
			return nil, fmt.Errorf("variable %q select options have type %T", definition.Key, definition.Options)
		}
		multiple := options.Multiple
		return &optionsJSON{Choices: encodeChoices(options.Choices), Multiple: &multiple}, nil
	case field.TypePhone:
		options, ok := definition.Options.(field.PhoneOptions)
		if definition.Options == nil {
			return &optionsJSON{}, nil
		}
		if !ok {
			return nil, fmt.Errorf("variable %q phone options have type %T", definition.Key, definition.Options)
		}
		pattern := options.Pattern
		return &optionsJSON{Pattern: &pattern}, nil
	case field.TypeFile:
		options, err := field.FileOptionsValue(definition.Options)
		if err != nil {
			return nil, err
		}
		return &optionsJSON{Storages: append([]filesystem.Code(nil), options.Storages...), MIMETypes: append([]string(nil), options.MIMETypes...)}, nil
	default:
		return nil, fmt.Errorf("unsupported variable type %q", definition.Type)
	}
}

func decodeOptions(code field.TypeCode, options *optionsJSON) (any, error) {
	switch code {
	case field.TypeString, field.TypeTextarea, field.TypeEmail, field.TypeCheckbox:
		if options != nil {
			return nil, errors.New("options are not supported")
		}
		return nil, nil
	case field.TypeInteger:
		result := field.IntegerOptions{}
		if options != nil && options.Step != nil {
			if math.Trunc(*options.Step) != *options.Step {
				return nil, errors.New("integer step is fractional")
			}
			step := int64(*options.Step)
			result.Step = &step
		}
		return result, nil
	case field.TypeFloat:
		result := field.FloatOptions{}
		if options != nil {
			result.Step = options.Step
		}
		return result, nil
	case field.TypeRadio:
		if options == nil {
			return nil, errors.New("radio options are missing")
		}
		return field.RadioOptions{Choices: decodeChoices(options.Choices)}, nil
	case field.TypeSelect:
		if options == nil {
			return nil, errors.New("select options are missing")
		}
		multiple := options.Multiple != nil && *options.Multiple
		return field.SelectOptions{Choices: decodeChoices(options.Choices), Multiple: multiple}, nil
	case field.TypePhone:
		result := field.PhoneOptions{}
		if options != nil && options.Pattern != nil {
			result.Pattern = *options.Pattern
		}
		return result, nil
	case field.TypeFile:
		if options == nil {
			return field.FileOptions{}, nil
		}
		return field.FileOptions{Storages: append([]filesystem.Code(nil), options.Storages...), MIMETypes: append([]string(nil), options.MIMETypes...)}, nil
	default:
		return nil, fmt.Errorf("unsupported variable type %q", code)
	}
}

func encodeChoices(items []field.Choice) []choiceJSON {
	result := make([]choiceJSON, len(items))
	for index, item := range items {
		result[index] = choiceJSON{Value: item.Value, Label: item.Label}
	}
	return result
}

func decodeChoices(items []choiceJSON) []field.Choice {
	result := make([]field.Choice, len(items))
	for index, item := range items {
		result[index] = field.Choice{Value: item.Value, Label: item.Label}
	}
	return result
}

func encodeJSON(value any) ([]byte, error) { return json.Marshal(value) }

func decodeJSON(raw []byte, target any) error { return json.Unmarshal(raw, target) }
