package forms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/vernal96/go-cms/kernel/security"
)

type ConfigField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

type ElementTypeMetadata struct {
	Code   ElementTypeCode `json:"code"`
	Label  string          `json:"label"`
	Fields []ConfigField   `json:"fields"`
}

type ElementType interface {
	Code() ElementTypeCode
	Metadata() ElementTypeMetadata
	ValidateConfig(json.RawMessage) error
}

type elementCatalog struct {
	types map[ElementTypeCode]ElementType
}

func newElementCatalog() (*elementCatalog, error) {
	items := []ElementType{
		jsonElementType{code: ElementText, label: "Текст", fields: []ConfigField{{Key: "content", Label: "Текст", Type: "textarea", Required: true}}},
		jsonElementType{code: ElementHeading, label: "Заголовок", fields: []ConfigField{{Key: "text", Label: "Заголовок", Type: "string", Required: true}, {Key: "level", Label: "Уровень", Type: "int", Required: true}}},
		jsonElementType{code: ElementImage, label: "Изображение", fields: []ConfigField{{Key: "file_id", Label: "Файл", Type: "file", Required: true}, {Key: "alt", Label: "Alt", Type: "string"}}},
		jsonElementType{code: ElementSubmitButton, label: "Кнопка отправки", fields: []ConfigField{{Key: "label", Label: "Текст кнопки", Type: "string", Required: true}}},
	}
	result := &elementCatalog{types: make(map[ElementTypeCode]ElementType, len(items))}
	for _, item := range items {
		if _, exists := result.types[item.Code()]; exists {
			return nil, fmt.Errorf("element type %q is duplicated", item.Code())
		}
		result.types[item.Code()] = item
	}
	return result, nil
}

func (c *elementCatalog) Type(code ElementTypeCode) (ElementType, bool) {
	item, exists := c.types[code]
	return item, exists
}

func (c *elementCatalog) Metadata() []ElementTypeMetadata {
	result := make([]ElementTypeMetadata, 0, len(c.types))
	for _, item := range c.types {
		result = append(result, item.Metadata())
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result
}

type jsonElementType struct {
	code   ElementTypeCode
	label  string
	fields []ConfigField
}

func (t jsonElementType) Code() ElementTypeCode { return t.code }
func (t jsonElementType) Metadata() ElementTypeMetadata {
	return ElementTypeMetadata{Code: t.code, Label: t.label, Fields: append([]ConfigField(nil), t.fields...)}
}
func (t jsonElementType) ValidateConfig(raw json.RawMessage) error {
	var values map[string]any
	if len(raw) == 0 || !json.Valid(raw) || json.Unmarshal(raw, &values) != nil {
		return fmt.Errorf("%w: element config is invalid", ErrInvalid)
	}
	for _, configField := range t.fields {
		value, exists := values[configField.Key]
		if configField.Required && (!exists || value == nil || value == "") {
			return fmt.Errorf("%w: element config field %q is required", ErrInvalid, configField.Key)
		}
	}
	return nil
}

type ActionTypeMetadata struct {
	Code       string        `json:"code"`
	Label      string        `json:"label"`
	Available  bool          `json:"available"`
	EditorCode string        `json:"editor_code,omitempty"`
	Fields     []ConfigField `json:"fields,omitempty"`
}

type ActionValidationContext struct {
	Actor   security.Actor
	Form    Form
	Fields  []FormField
	Trigger Trigger
}

type ActionExecutionContext struct {
	Execution ActionExecution
	Result    Result
	Values    []ResultValue
	Uploads   UploadAccessor
}

type ActionExecutionResult struct {
	ExternalReference string
}

type ActionType interface {
	Code() string
	Metadata() ActionTypeMetadata
	ValidateConfig(context.Context, ActionValidationContext, json.RawMessage) error
	Execute(context.Context, ActionExecutionContext, json.RawMessage) (ActionExecutionResult, error)
}

type ActionRegistrar interface {
	RegisterActionType(ActionType) error
}

type actionRegistry struct {
	mu     sync.RWMutex
	types  map[string]ActionType
	sealed bool
}

func newActionRegistry() *actionRegistry {
	return &actionRegistry{types: make(map[string]ActionType)}
}

func (r *actionRegistry) Register(actionType ActionType) error {
	if actionType == nil {
		return errors.New("Forms action type is nil")
	}
	code := strings.TrimSpace(actionType.Code())
	if code == "" || code != actionType.Code() {
		return errors.New("Forms action type code is invalid")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sealed {
		return errors.New("Forms action registry is sealed")
	}
	if _, exists := r.types[code]; exists {
		return fmt.Errorf("Forms action type %q is already registered", code)
	}
	metadata := actionType.Metadata()
	if metadata.Code != code || strings.TrimSpace(metadata.Label) == "" {
		return fmt.Errorf("Forms action type %q metadata is invalid", code)
	}
	r.types[code] = actionType
	return nil
}

func (r *actionRegistry) Seal() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sealed {
		return errors.New("Forms action registry is already sealed")
	}
	r.sealed = true
	return nil
}

func (r *actionRegistry) Type(code string) (ActionType, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, exists := r.types[code]
	return item, exists
}

func (r *actionRegistry) Metadata() []ActionTypeMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]ActionTypeMetadata, 0, len(r.types))
	for _, item := range r.types {
		metadata := item.Metadata()
		metadata.Available = true
		result = append(result, metadata)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Code < result[j].Code })
	return result
}

type UploadAccessor interface {
	Metadata(fieldCode string) []ResultUpload
	Open(context.Context, string, int) (io.ReadCloser, error)
}

type ActionError struct {
	Code      string
	Retryable bool
	Err       error
}

func (e *ActionError) Error() string {
	if e == nil || e.Err == nil {
		return "Forms action failed"
	}
	return e.Err.Error()
}

func (e *ActionError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
