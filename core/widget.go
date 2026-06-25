package core

import (
	"context"

	"github.com/vernal96/go-cms/core/fields"
)

type WidgetCode string
type WidgetTemplateCode string

const WidgetTemplateDefault WidgetTemplateCode = "default"

type Widget interface {
	Code() WidgetCode
	Name() string
	Params() []WidgetParamDefinition
	Render(ctx context.Context, params WidgetParams) (WidgetResult, error)
}

type WidgetParams map[string]any

type WidgetResult struct {
	Data map[string]any
	HTML string `json:"-"`
}

type WidgetTemplate interface {
	Code() WidgetTemplateCode
	Name() string
}

type WidgetParamCode string

type WidgetParamDefinition struct {
	Code        WidgetParamCode
	Name        string
	Field       fields.FieldType
	Required    bool
	Default     any
	Description string
}
