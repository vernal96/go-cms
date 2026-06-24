package controllers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/vernal96/go-cms/core"
)

const ResourceRoutePath = "/_cms/resource"

type ResourceController struct{}

func NewResourceController() *ResourceController {
	return &ResourceController{}
}

func (c *ResourceController) Routes() []core.Route {
	return []core.Route{
		{
			Method:  core.RouteMethodGet,
			Path:    ResourceRoutePath,
			Handler: c.resource,
		},
	}
}

func (c *ResourceController) resource(
	ctx context.Context,
	runtime *core.SiteRuntime,
	request *http.Request,
) (any, error) {
	if runtime == nil {
		return nil, errors.New("site runtime is nil")
	}

	resourcePath := request.URL.Query().Get("path")
	if resourcePath == "" {
		resourcePath = "/"
	}
	if !strings.HasPrefix(resourcePath, "/") {
		return nil, errors.New("resource path must start with /")
	}

	resource, err := runtime.App().Resources().FindByPath(ctx, runtime.Site().ID, resourcePath)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"resource": resource,
	}, nil
}

var _ core.Controller = (*ResourceController)(nil)
