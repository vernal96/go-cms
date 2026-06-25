package widgets

import (
	"context"
	"fmt"
	"html"

	"github.com/vernal96/go-cms/core"
	modulefields "github.com/vernal96/go-cms/core/modules/core/fields"
)

const (
	TextWidgetCode      core.WidgetCode      = "text"
	TextWidgetParamText core.WidgetParamCode = "text"
)

type TextWidget struct{}

func NewTextWidget() *TextWidget {
	return &TextWidget{}
}

func (w *TextWidget) Code() core.WidgetCode {
	return TextWidgetCode
}

func (w *TextWidget) Name() string {
	return "Text"
}

func (w *TextWidget) Params() []core.WidgetParamDefinition {
	return []core.WidgetParamDefinition{
		{
			Code:  TextWidgetParamText,
			Name:  "Text",
			Field: modulefields.NewTextFieldType(),
		},
	}
}

func (w *TextWidget) Render(
	ctx context.Context,
	params core.WidgetParams,
) (core.WidgetResult, error) {
	if err := ctx.Err(); err != nil {
		return core.WidgetResult{}, err
	}

	value, exists := params[string(TextWidgetParamText)]
	if !exists || value == nil {
		return core.WidgetResult{}, nil
	}

	text, ok := value.(string)
	if !ok {
		text = fmt.Sprint(value)
	}

	return core.WidgetResult{
		HTML: html.EscapeString(text),
	}, nil
}

var _ core.Widget = (*TextWidget)(nil)
