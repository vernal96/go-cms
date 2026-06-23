package core

import "context"

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
}

type WidgetTemplate interface {
	Code() WidgetTemplateCode
	Name() string
}
