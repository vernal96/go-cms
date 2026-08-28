package postgres

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/forms"
)

type optionsJSON struct {
	Step        *float64       `json:"step,omitempty"`
	Choices     []field.Choice `json:"choices,omitempty"`
	Multiple    *bool          `json:"multiple,omitempty"`
	Pattern     *string        `json:"pattern,omitempty"`
	MIMETypes   []string       `json:"mime_types,omitempty"`
	MaxFileSize *int64         `json:"max_file_size,omitempty"`
	MaxFiles    *int           `json:"max_files,omitempty"`
	Provider    *string        `json:"provider,omitempty"`
	Text        *string        `json:"text,omitempty"`
	URL         *string        `json:"url,omitempty"`
}

func encodeFieldOptions(item forms.FormField) ([]byte, error) {
	var options *optionsJSON
	switch item.Type {
	case field.TypeString, field.TypeTextarea, field.TypeEmail, field.TypeCheckbox, field.TypeJSON:
		if item.Options != nil {
			return nil, fmt.Errorf("field %q has unsupported options", item.Code)
		}
	case field.TypeInteger:
		value, ok := item.Options.(field.IntegerOptions)
		if item.Options == nil {
			ok = true
		}
		if !ok {
			return nil, fmt.Errorf("field %q integer options have type %T", item.Code, item.Options)
		}
		options = &optionsJSON{}
		if value.Step != nil {
			step := float64(*value.Step)
			options.Step = &step
		}
	case field.TypeFloat:
		value, ok := item.Options.(field.FloatOptions)
		if item.Options == nil {
			ok = true
		}
		if !ok {
			return nil, fmt.Errorf("field %q float options have type %T", item.Code, item.Options)
		}
		options = &optionsJSON{Step: value.Step}
	case field.TypeRadio:
		value, ok := item.Options.(field.RadioOptions)
		if !ok {
			return nil, fmt.Errorf("field %q radio options have type %T", item.Code, item.Options)
		}
		options = &optionsJSON{Choices: value.Choices}
	case field.TypeSelect:
		value, ok := item.Options.(field.SelectOptions)
		if !ok {
			return nil, fmt.Errorf("field %q select options have type %T", item.Code, item.Options)
		}
		multiple := value.Multiple
		options = &optionsJSON{Choices: value.Choices, Multiple: &multiple}
	case field.TypePhone:
		value, ok := item.Options.(field.PhoneOptions)
		if item.Options == nil {
			ok = true
		}
		if !ok {
			return nil, fmt.Errorf("field %q phone options have type %T", item.Code, item.Options)
		}
		pattern := value.Pattern
		options = &optionsJSON{Pattern: &pattern}
	case field.TypeFile:
		value, ok := item.Options.(field.FileOptions)
		if item.Options == nil {
			ok = true
		}
		if !ok {
			return nil, fmt.Errorf("field %q file options have type %T", item.Code, item.Options)
		}
		return json.Marshal(value)
	case forms.FieldTypeCaptcha:
		value, ok := item.Options.(forms.CaptchaOptions)
		if item.Options == nil {
			ok = true
		}
		if !ok {
			return nil, fmt.Errorf("field %q CAPTCHA options have type %T", item.Code, item.Options)
		}
		provider := value.Provider
		options = &optionsJSON{Provider: &provider}
	case forms.FieldTypeConsent:
		value, ok := item.Options.(forms.ConsentOptions)
		if item.Options == nil {
			ok = true
		}
		if !ok {
			return nil, fmt.Errorf("field %q consent options have type %T", item.Code, item.Options)
		}
		text, url := value.Text, value.URL
		options = &optionsJSON{Text: &text, URL: &url}
	case forms.FieldTypeUpload:
		value, ok := item.Options.(forms.UploadOptions)
		if item.Options == nil {
			ok = true
		}
		if !ok {
			return nil, fmt.Errorf("field %q upload options have type %T", item.Code, item.Options)
		}
		multiple, maxSize, maxFiles := value.Multiple, value.MaxFileSize, value.MaxFiles
		options = &optionsJSON{MIMETypes: value.MIMETypes, Multiple: &multiple, MaxFileSize: &maxSize, MaxFiles: &maxFiles}
	default:
		if item.Options == nil {
			return nil, nil
		}
		return json.Marshal(item.Options)
	}
	if options == nil {
		return nil, nil
	}
	return json.Marshal(options)
}

func decodeFieldOptions(code field.TypeCode, raw []byte) (any, error) {
	var options *optionsJSON
	if len(raw) > 0 {
		options = &optionsJSON{}
		if err := json.Unmarshal(raw, options); err != nil {
			return nil, err
		}
	}
	switch code {
	case field.TypeString, field.TypeTextarea, field.TypeEmail, field.TypeCheckbox, field.TypeJSON:
		if options != nil {
			return nil, errors.New("options are unsupported")
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
		return field.RadioOptions{Choices: options.Choices}, nil
	case field.TypeSelect:
		if options == nil {
			return nil, errors.New("select options are missing")
		}
		return field.SelectOptions{Choices: options.Choices, Multiple: options.Multiple != nil && *options.Multiple}, nil
	case field.TypePhone:
		result := field.PhoneOptions{}
		if options != nil && options.Pattern != nil {
			result.Pattern = *options.Pattern
		}
		return result, nil
	case field.TypeFile:
		if len(raw) == 0 {
			return field.FileOptions{}, nil
		}
		var result field.FileOptions
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, err
		}
		return result, nil
	case forms.FieldTypeCaptcha:
		result := forms.CaptchaOptions{}
		if options != nil && options.Provider != nil {
			result.Provider = *options.Provider
		}
		return result, nil
	case forms.FieldTypeConsent:
		result := forms.ConsentOptions{}
		if options != nil {
			if options.Text != nil {
				result.Text = *options.Text
			}
			if options.URL != nil {
				result.URL = *options.URL
			}
		}
		return result, nil
	case forms.FieldTypeUpload:
		result := forms.UploadOptions{}
		if options != nil {
			result.MIMETypes = options.MIMETypes
			result.Multiple = options.Multiple != nil && *options.Multiple
			if options.MaxFileSize != nil {
				result.MaxFileSize = *options.MaxFileSize
			}
			if options.MaxFiles != nil {
				result.MaxFiles = *options.MaxFiles
			}
		}
		return result, nil
	default:
		if len(raw) == 0 {
			return nil, nil
		}
		var result any
		if err := json.Unmarshal(raw, &result); err != nil {
			return nil, err
		}
		return result, nil
	}
}

func encodeJSON(value any) ([]byte, error)    { return json.Marshal(value) }
func decodeJSON(raw []byte, target any) error { return json.Unmarshal(raw, target) }
