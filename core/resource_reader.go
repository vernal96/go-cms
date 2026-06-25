package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type ResourceReader struct{}

func NewResourceReader() *ResourceReader {
	return &ResourceReader{}
}

func (r *ResourceReader) ReadByPath(
	ctx context.Context,
	runtime *SiteRuntime,
	path string,
) (ResourceData, error) {
	if err := ctx.Err(); err != nil {
		return ResourceData{}, err
	}

	if runtime == nil {
		return ResourceData{}, errors.New("site runtime is nil")
	}

	if path == "" {
		return ResourceData{}, errors.New("resource path is empty")
	}

	if !strings.HasPrefix(path, "/") {
		return ResourceData{}, errors.New("resource path must start with /")
	}

	resource, err := runtime.App().Resources().FindByPath(
		ctx,
		runtime.Site().ID,
		path,
	)
	if err != nil {
		return ResourceData{}, err
	}

	if _, exists := runtime.Registry().ResourceTypes().Get(resource.Type); !exists {
		return ResourceData{}, fmt.Errorf("resource type %q is not registered", resource.Type)
	}

	resourceTemplate := ResourceTemplateCode(resource.Template)
	if _, exists := runtime.Registry().ResourceTemplates().Get(
		resource.Type,
		resourceTemplate,
	); !exists {
		return ResourceData{}, fmt.Errorf(
			"resource template %q for resource type %q is not registered",
			resourceTemplate,
			resource.Type,
		)
	}

	fields := runtime.Registry().ResourceFields().AllForTemplate(
		resource.Type,
		resourceTemplate,
	)

	values, err := runtime.App().ResourceFieldValues().FindByResourceID(ctx, resource.ID)
	if err != nil {
		return ResourceData{}, err
	}

	widgets, err := runtime.App().WidgetInstances().FindForResource(ctx, resource)
	if err != nil {
		return ResourceData{}, err
	}

	return ResourceData{
		Resource: resource,
		Fields:   fields,
		Values:   values,
		Widgets:  widgets,
	}, nil
}
