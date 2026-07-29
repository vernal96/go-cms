package widgets

import (
	"context"
	"errors"

	"github.com/vernal96/go-cms/kernel/modules/core/widget"
)

const ContentWidgetCode widget.Code = "content"

type contentWidget struct{}

func (contentWidget) Definition() widget.Definition {
	return widget.Definition{
		Code:        ContentWidgetCode,
		Label:       "Content",
		Description: "Returns the resource content",
	}
}

func (contentWidget) New(
	map[string]any,
) (widget.Instance, error) {
	return contentWidgetInstance{}, nil
}

type contentWidgetInstance struct{}

func (contentWidgetInstance) Render(
	ctx context.Context,
	input widget.RenderInput,
) (map[string]any, error) {
	if ctx == nil {
		return nil, errors.New("widget render context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return map[string]any{
		"content": input.Resource.Content,
	}, nil
}

var _ widget.Widget = contentWidget{}
var _ widget.Instance = contentWidgetInstance{}
