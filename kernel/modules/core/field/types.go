package field

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

func StandardTypes() []Type {
	return []Type{
		stringType{code: TypeString},
		integerType{},
		floatType{},
		boolType{},
		choiceType{code: TypeRadio},
		choiceType{code: TypeSelect},
		stringType{code: TypeTextarea},
		stringType{code: TypeEmail, rules: []string{"email"}},
		phoneType{},
	}
}

type stringType struct {
	code  TypeCode
	rules []string
}

func (t stringType) Code() TypeCode {
	return t.code
}

func (t stringType) Compile(options any) (ValueType, error) {
	if options != nil {
		return nil, fmt.Errorf("%s field does not support options", t.code)
	}

	return stringValue{rules: append([]string(nil), t.rules...)}, nil
}

type stringValue struct {
	rules []string
}

func (v stringValue) Normalize(value any) (any, error) {
	result, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", value)
	}

	return result, nil
}

func (stringValue) Empty(value any) bool {
	result, ok := value.(string)
	return ok && result == ""
}

func (stringValue) Validate(any) error {
	return nil
}

func (v stringValue) Rules() []string {
	return append([]string(nil), v.rules...)
}

func (stringValue) Example() any {
	return "example"
}

type integerType struct{}

func (integerType) Code() TypeCode {
	return TypeInteger
}

func (integerType) Compile(options any) (ValueType, error) {
	config, err := integerOptions(options)
	if err != nil {
		return nil, err
	}
	if config.Step != nil && *config.Step <= 0 {
		return nil, errors.New("integer step must be greater than zero")
	}

	return integerValue{}, nil
}

type integerValue struct{}

func (integerValue) Normalize(value any) (any, error) {
	result, ok := normalizeInteger(value)
	if !ok {
		return nil, fmt.Errorf("expected integer, got %T", value)
	}

	return result, nil
}

func (integerValue) Empty(any) bool {
	return false
}

func (integerValue) Validate(any) error {
	return nil
}

func (integerValue) Rules() []string {
	return nil
}

func (integerValue) Example() any {
	return int64(1)
}

type floatType struct{}

func (floatType) Code() TypeCode {
	return TypeFloat
}

func (floatType) Compile(options any) (ValueType, error) {
	config, err := floatOptions(options)
	if err != nil {
		return nil, err
	}
	if config.Step != nil &&
		(*config.Step <= 0 || math.IsNaN(*config.Step) || math.IsInf(*config.Step, 0)) {
		return nil, errors.New("float step must be finite and greater than zero")
	}

	return floatValue{}, nil
}

type floatValue struct{}

func (floatValue) Normalize(value any) (any, error) {
	result, ok := normalizeFloat(value)
	if !ok {
		return nil, fmt.Errorf("expected number, got %T", value)
	}

	return result, nil
}

func (floatValue) Empty(any) bool {
	return false
}

func (floatValue) Validate(any) error {
	return nil
}

func (floatValue) Rules() []string {
	return nil
}

func (floatValue) Example() any {
	return float64(1)
}

type boolType struct{}

func (boolType) Code() TypeCode {
	return TypeCheckbox
}

func (boolType) Compile(options any) (ValueType, error) {
	if options != nil {
		return nil, errors.New("checkbox field does not support options")
	}

	return boolValue{}, nil
}

type boolValue struct{}

func (boolValue) Normalize(value any) (any, error) {
	result, ok := value.(bool)
	if !ok {
		return nil, fmt.Errorf("expected bool, got %T", value)
	}

	return result, nil
}

func (boolValue) Empty(any) bool {
	return false
}

func (boolValue) Validate(any) error {
	return nil
}

func (boolValue) Rules() []string {
	return nil
}

func (boolValue) Example() any {
	return false
}

type choiceType struct {
	code TypeCode
}

func (t choiceType) Code() TypeCode {
	return t.code
}

func (t choiceType) Compile(options any) (ValueType, error) {
	var (
		choices  []Choice
		multiple bool
		err      error
	)

	switch t.code {
	case TypeRadio:
		config, configErr := radioOptions(options)
		if configErr != nil {
			err = configErr
		} else {
			choices = config.Choices
		}

	case TypeSelect:
		config, configErr := selectOptions(options)
		if configErr != nil {
			err = configErr
		} else {
			choices = config.Choices
			multiple = config.Multiple
		}

	default:
		err = fmt.Errorf("unsupported choice type %q", t.code)
	}
	if err != nil {
		return nil, err
	}

	allowed, err := compileChoices(choices)
	if err != nil {
		return nil, err
	}

	return choiceValue{
		allowed:  allowed,
		multiple: multiple,
	}, nil
}

type choiceValue struct {
	allowed  map[string]struct{}
	multiple bool
}

func (v choiceValue) Normalize(value any) (any, error) {
	if !v.multiple {
		result, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", value)
		}

		return result, nil
	}

	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...), nil

	case []any:
		result := make([]string, len(typed))
		for index, item := range typed {
			stringValue, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf(
					"expected string at index %d, got %T",
					index,
					item,
				)
			}
			result[index] = stringValue
		}
		return result, nil

	default:
		return nil, fmt.Errorf("expected string slice, got %T", value)
	}
}

func (v choiceValue) Empty(value any) bool {
	if v.multiple {
		result, ok := value.([]string)
		return ok && len(result) == 0
	}

	result, ok := value.(string)
	return ok && result == ""
}

func (v choiceValue) Validate(value any) error {
	if !v.multiple {
		result := value.(string)
		if _, exists := v.allowed[result]; !exists {
			return RuleError{Rule: "oneof"}
		}
		return nil
	}

	for _, item := range value.([]string) {
		if _, exists := v.allowed[item]; !exists {
			return RuleError{Rule: "oneof", Param: item}
		}
	}

	return nil
}

func (choiceValue) Rules() []string {
	return nil
}

func (v choiceValue) Example() any {
	if v.multiple {
		return []string{"example"}
	}

	return "example"
}

type phoneType struct{}

func (phoneType) Code() TypeCode {
	return TypePhone
}

func (phoneType) Compile(options any) (ValueType, error) {
	config, err := phoneOptions(options)
	if err != nil {
		return nil, err
	}

	result := phoneValue{}
	if config.Pattern == "" {
		result.rules = []string{"e164"}
		return result, nil
	}

	pattern, err := regexp.Compile(config.Pattern)
	if err != nil {
		return nil, fmt.Errorf("compile phone pattern: %w", err)
	}
	result.pattern = pattern
	return result, nil
}

type phoneValue struct {
	pattern *regexp.Regexp
	rules   []string
}

func (phoneValue) Normalize(value any) (any, error) {
	result, ok := value.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", value)
	}

	return result, nil
}

func (phoneValue) Empty(value any) bool {
	result, ok := value.(string)
	return ok && result == ""
}

func (v phoneValue) Validate(value any) error {
	if v.pattern == nil {
		return nil
	}
	if !v.pattern.MatchString(value.(string)) {
		return RuleError{Rule: "pattern", Param: v.pattern.String()}
	}

	return nil
}

func (v phoneValue) Rules() []string {
	return append([]string(nil), v.rules...)
}

func (phoneValue) Example() any {
	return "+79991234567"
}

func integerOptions(options any) (IntegerOptions, error) {
	switch typed := options.(type) {
	case nil:
		return IntegerOptions{}, nil
	case IntegerOptions:
		return typed, nil
	case *IntegerOptions:
		if typed == nil {
			return IntegerOptions{}, errors.New("integer options are nil")
		}
		return *typed, nil
	default:
		return IntegerOptions{}, fmt.Errorf(
			"invalid integer options type %T",
			options,
		)
	}
}

func floatOptions(options any) (FloatOptions, error) {
	switch typed := options.(type) {
	case nil:
		return FloatOptions{}, nil
	case FloatOptions:
		return typed, nil
	case *FloatOptions:
		if typed == nil {
			return FloatOptions{}, errors.New("float options are nil")
		}
		return *typed, nil
	default:
		return FloatOptions{}, fmt.Errorf(
			"invalid float options type %T",
			options,
		)
	}
}

func radioOptions(options any) (RadioOptions, error) {
	switch typed := options.(type) {
	case RadioOptions:
		return typed, nil
	case *RadioOptions:
		if typed == nil {
			return RadioOptions{}, errors.New("radio options are nil")
		}
		return *typed, nil
	default:
		return RadioOptions{}, fmt.Errorf(
			"invalid radio options type %T",
			options,
		)
	}
}

func selectOptions(options any) (SelectOptions, error) {
	switch typed := options.(type) {
	case SelectOptions:
		return typed, nil
	case *SelectOptions:
		if typed == nil {
			return SelectOptions{}, errors.New("select options are nil")
		}
		return *typed, nil
	default:
		return SelectOptions{}, fmt.Errorf(
			"invalid select options type %T",
			options,
		)
	}
}

func phoneOptions(options any) (PhoneOptions, error) {
	switch typed := options.(type) {
	case nil:
		return PhoneOptions{}, nil
	case PhoneOptions:
		return typed, nil
	case *PhoneOptions:
		if typed == nil {
			return PhoneOptions{}, errors.New("phone options are nil")
		}
		return *typed, nil
	default:
		return PhoneOptions{}, fmt.Errorf(
			"invalid phone options type %T",
			options,
		)
	}
}

func compileChoices(choices []Choice) (map[string]struct{}, error) {
	if len(choices) == 0 {
		return nil, errors.New("choices are empty")
	}

	result := make(map[string]struct{}, len(choices))
	for index, choice := range choices {
		if choice.Value == "" || strings.TrimSpace(choice.Value) != choice.Value {
			return nil, fmt.Errorf(
				"choice at index %d has invalid value %q",
				index,
				choice.Value,
			)
		}
		if choice.Label == "" || strings.TrimSpace(choice.Label) != choice.Label {
			return nil, fmt.Errorf(
				"choice %q has invalid label %q",
				choice.Value,
				choice.Label,
			)
		}
		if _, exists := result[choice.Value]; exists {
			return nil, fmt.Errorf("duplicate choice value %q", choice.Value)
		}
		result[choice.Value] = struct{}{}
	}

	return result, nil
}

func normalizeInteger(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int8:
		return int64(typed), true
	case int16:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		if uint64(typed) > math.MaxInt64 {
			return 0, false
		}
		return int64(typed), true
	case uint8:
		return int64(typed), true
	case uint16:
		return int64(typed), true
	case uint32:
		return int64(typed), true
	case uint64:
		if typed > math.MaxInt64 {
			return 0, false
		}
		return int64(typed), true
	case float32:
		return integralFloat(float64(typed))
	case float64:
		return integralFloat(typed)
	case json.Number:
		if result, err := typed.Int64(); err == nil {
			return result, true
		}
		result, err := typed.Float64()
		if err != nil {
			return 0, false
		}
		return integralFloat(result)
	default:
		return 0, false
	}
}

func integralFloat(value float64) (int64, bool) {
	const (
		minInt64Float     = -9223372036854775808.0
		maxInt64Exclusive = 9223372036854775808.0
	)

	if math.IsNaN(value) ||
		math.IsInf(value, 0) ||
		math.Trunc(value) != value ||
		value < minInt64Float ||
		value >= maxInt64Exclusive {
		return 0, false
	}

	return int64(value), true
}

func normalizeFloat(value any) (float64, bool) {
	var result float64

	switch typed := value.(type) {
	case int:
		result = float64(typed)
	case int8:
		result = float64(typed)
	case int16:
		result = float64(typed)
	case int32:
		result = float64(typed)
	case int64:
		result = float64(typed)
	case uint:
		result = float64(typed)
	case uint8:
		result = float64(typed)
	case uint16:
		result = float64(typed)
	case uint32:
		result = float64(typed)
	case uint64:
		result = float64(typed)
	case float32:
		result = float64(typed)
	case float64:
		result = typed
	case json.Number:
		parsed, err := strconv.ParseFloat(string(typed), 64)
		if err != nil {
			return 0, false
		}
		result = parsed
	default:
		return 0, false
	}

	if math.IsNaN(result) || math.IsInf(result, 0) {
		return 0, false
	}

	return result, true
}

func inputEmpty(value any) bool {
	if value == nil {
		return true
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Array, reflect.Slice, reflect.String:
		return reflected.Len() == 0
	default:
		return false
	}
}
