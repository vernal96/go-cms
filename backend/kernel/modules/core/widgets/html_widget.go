package widgets

import (
	"context"
	"errors"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
)

var HTML = widget.NewRef("html")

type htmlWidget struct{}

func (htmlWidget) Definition() widget.Definition {
	return widget.Definition{
		Reference: HTML, Label: "Текстовый контент", Description: "Редактируемый HTML-контент",
		Fields: []field.Definition{{Key: "html", Type: field.TypeString, Label: "HTML", Editor: "html"}},
	}
}
func (htmlWidget) New(values map[string]any) (widget.Instance, error) {
	html, _ := values["html"].(string)
	return htmlWidgetInstance{html: html}, nil
}

type htmlWidgetInstance struct{ html string }

func (i htmlWidgetInstance) Render(ctx context.Context, _ widget.RenderInput) (map[string]any, error) {
	if ctx == nil {
		return nil, errors.New("widget render context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return map[string]any{"html": i.html}, nil
}

var _ widget.Widget = htmlWidget{}
var _ widget.Instance = htmlWidgetInstance{}
