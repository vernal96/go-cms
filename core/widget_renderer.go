package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type WidgetRenderer struct{}

func NewWidgetRenderer() *WidgetRenderer {
	return &WidgetRenderer{}
}

func (r *WidgetRenderer) RenderArea(
	ctx context.Context,
	runtime *SiteRuntime,
	data ResourceData,
	area WidgetArea,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if runtime == nil {
		return "", errors.New("site runtime is nil")
	}

	var output strings.Builder
	for _, instance := range data.Widgets {
		if instance.Area != area {
			continue
		}

		widget, exists := runtime.Registry().Widgets().Get(instance.Widget)
		if !exists {
			return "", fmt.Errorf("widget %q is not registered", instance.Widget)
		}

		if instance.Template != "" && instance.Template != WidgetTemplateDefault {
			if _, exists := runtime.Registry().WidgetTemplates().Get(
				instance.Widget,
				instance.Template,
			); !exists {
				return "", fmt.Errorf(
					"widget template %q for widget %q is not registered",
					instance.Template,
					instance.Widget,
				)
			}
		}

		result, err := widget.Render(ctx, instance.Params)
		if err != nil {
			return "", fmt.Errorf("render widget %q: %w", instance.Widget, err)
		}

		output.WriteString(result.HTML)
	}

	return output.String(), nil
}
