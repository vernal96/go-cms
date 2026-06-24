package core

import (
	"context"
	"errors"
	"fmt"
)

type ResourceRenderer struct{}

func NewResourceRenderer() *ResourceRenderer {
	return &ResourceRenderer{}
}

func (r *ResourceRenderer) Render(
	ctx context.Context,
	runtime *SiteRuntime,
	data ResourceData,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if runtime == nil {
		return "", errors.New("site runtime is nil")
	}

	resource := data.Resource
	if resource.ID <= 0 {
		return "", errors.New("resource id must be positive")
	}

	if _, exists := runtime.Registry().ResourceTypes().Get(resource.Type); !exists {
		return "", fmt.Errorf("resource type %q is not registered", resource.Type)
	}

	resourceTemplate := ResourceTemplateCode(resource.Template)
	if _, exists := runtime.Registry().ResourceTemplates().Get(
		resource.Type,
		resourceTemplate,
	); !exists {
		return "", fmt.Errorf(
			"resource template %q for resource type %q is not registered",
			resourceTemplate,
			resource.Type,
		)
	}

	renderer, exists := runtime.Registry().ResourceTemplateRenderers().Get(
		resource.Type,
		resourceTemplate,
	)
	if !exists {
		return "", fmt.Errorf(
			"resource template renderer for resource type %q and template %q is not registered",
			resource.Type,
			resourceTemplate,
		)
	}

	return renderer.Render(ctx, runtime, data)
}
