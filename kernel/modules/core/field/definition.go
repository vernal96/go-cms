package field

import (
	"fmt"
	"reflect"
)

type TypeCode string

const (
	TypeString   TypeCode = "string"
	TypeInteger  TypeCode = "int"
	TypeFloat    TypeCode = "float"
	TypeCheckbox TypeCode = "checkbox"
	TypeRadio    TypeCode = "radio"
	TypeSelect   TypeCode = "select"
	TypeTextarea TypeCode = "textarea"
	TypeEmail    TypeCode = "email"
	TypePhone    TypeCode = "phone"
)

type Definition struct {
	Key      string
	Type     TypeCode
	Label    string
	Required *bool
	Rules    []string
	Options  any
}

type IntegerOptions struct {
	Step *int64
}

type FloatOptions struct {
	Step *float64
}

type Choice struct {
	Value string
	Label string
}

type RadioOptions struct {
	Choices []Choice
}

type SelectOptions struct {
	Choices  []Choice
	Multiple bool
}

type PhoneOptions struct {
	Pattern string
}

type Type interface {
	Code() TypeCode
	Compile(options any) (ValueType, error)
}

type ValueType interface {
	Normalize(any) (any, error)
	Empty(any) bool
	Validate(any) error
	Rules() []string
	Example() any
}

type TypeResolver interface {
	FieldType(TypeCode) (Type, bool)
}

type RuleError struct {
	Rule  string
	Param string
}

func (e RuleError) Error() string {
	if e.Param == "" {
		return e.Rule
	}

	return fmt.Sprintf("%s=%s", e.Rule, e.Param)
}

type ValidationError struct {
	Key   string
	Rule  string
	Param string
}

type ValidationErrors []ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "field validation failed"
	}

	first := e[0]
	if first.Param == "" {
		return fmt.Sprintf(
			"field %q failed validation rule %q",
			first.Key,
			first.Rule,
		)
	}

	return fmt.Sprintf(
		"field %q failed validation rule %q with parameter %q",
		first.Key,
		first.Rule,
		first.Param,
	)
}

func CloneDefinitions(source []Definition) []Definition {
	if source == nil {
		return nil
	}

	result := make([]Definition, len(source))
	for index, definition := range source {
		result[index] = definition
		result[index].Rules = append([]string(nil), definition.Rules...)
		result[index].Options = cloneOptions(definition.Options)

		if definition.Required != nil {
			required := *definition.Required
			result[index].Required = &required
		}
	}

	return result
}

func cloneOptions(options any) any {
	switch typed := options.(type) {
	case RadioOptions:
		typed.Choices = append([]Choice(nil), typed.Choices...)
		return typed

	case *RadioOptions:
		if typed == nil {
			return (*RadioOptions)(nil)
		}
		result := *typed
		result.Choices = append([]Choice(nil), typed.Choices...)
		return &result

	case SelectOptions:
		typed.Choices = append([]Choice(nil), typed.Choices...)
		return typed

	case *SelectOptions:
		if typed == nil {
			return (*SelectOptions)(nil)
		}
		result := *typed
		result.Choices = append([]Choice(nil), typed.Choices...)
		return &result

	case IntegerOptions:
		if typed.Step != nil {
			step := *typed.Step
			typed.Step = &step
		}
		return typed

	case *IntegerOptions:
		if typed == nil {
			return (*IntegerOptions)(nil)
		}
		result := *typed
		if typed.Step != nil {
			step := *typed.Step
			result.Step = &step
		}
		return &result

	case FloatOptions:
		if typed.Step != nil {
			step := *typed.Step
			typed.Step = &step
		}
		return typed

	case *FloatOptions:
		if typed == nil {
			return (*FloatOptions)(nil)
		}
		result := *typed
		if typed.Step != nil {
			step := *typed.Step
			result.Step = &step
		}
		return &result

	case PhoneOptions:
		return typed

	case *PhoneOptions:
		if typed == nil {
			return (*PhoneOptions)(nil)
		}
		result := *typed
		return &result

	default:
		return typed
	}
}

func nilInterface(value any) bool {
	if value == nil {
		return true
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
