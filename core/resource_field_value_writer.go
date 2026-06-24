package core

import (
	"context"
	"errors"
	"fmt"
)

type ResourceFieldValueWriter struct{}

func NewResourceFieldValueWriter() *ResourceFieldValueWriter {
	return &ResourceFieldValueWriter{}
}

func (w *ResourceFieldValueWriter) Save(
	ctx context.Context,
	runtime *SiteRuntime,
	resource Resource,
	field ResourceFieldCode,
	value any,
) (ResourceFieldValue, error) {
	if err := ctx.Err(); err != nil {
		return ResourceFieldValue{}, err
	}

	if runtime == nil {
		return ResourceFieldValue{}, errors.New("site runtime is nil")
	}

	if resource.ID <= 0 {
		return ResourceFieldValue{}, errors.New("resource id must be positive")
	}

	if field == "" {
		return ResourceFieldValue{}, errors.New("resource field is empty")
	}

	if _, exists := runtime.Registry().ResourceTypes().Get(resource.Type); !exists {
		return ResourceFieldValue{}, fmt.Errorf(
			"resource type %q is not registered",
			resource.Type,
		)
	}

	resourceTemplate := ResourceTemplateCode(resource.Template)
	if _, exists := runtime.Registry().ResourceTemplates().Get(
		resource.Type,
		resourceTemplate,
	); !exists {
		return ResourceFieldValue{}, fmt.Errorf(
			"resource template %q for resource type %q is not registered",
			resourceTemplate,
			resource.Type,
		)
	}

	if _, exists := runtime.Registry().ResourceFields().Get(
		resource.Type,
		resourceTemplate,
		field,
	); !exists {
		return ResourceFieldValue{}, fmt.Errorf(
			"resource field %q for resource type %q and template %q is not registered",
			field,
			resource.Type,
			resourceTemplate,
		)
	}

	return runtime.App().ResourceFieldValues().Save(ctx, ResourceFieldValue{
		ResourceID: resource.ID,
		Field:      field,
		Value:      value,
	})
}
