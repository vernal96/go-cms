package controllers

import (
	"context"
	"net/http"

	"github.com/vernal96/go-cms/core"
)

const PageRoutePath = "/"

type PageController struct {
	reader   *core.ResourceReader
	renderer *core.ResourceRenderer
}

func NewPageController() *PageController {
	return &PageController{
		reader:   core.NewResourceReader(),
		renderer: core.NewResourceRenderer(),
	}
}

func (c *PageController) Routes() []core.Route {
	return []core.Route{
		{
			Method:  core.RouteMethodGet,
			Path:    PageRoutePath,
			Handler: c.page,
		},
	}
}

func (c *PageController) page(
	ctx context.Context,
	runtime *core.SiteRuntime,
	request *http.Request,
) (any, error) {
	resourcePath := request.URL.Path
	if resourcePath == "" {
		resourcePath = "/"
	}

	data, err := c.reader.ReadByPath(ctx, runtime, resourcePath)
	if err != nil {
		return nil, err
	}

	html, err := c.renderer.Render(ctx, runtime, data)
	if err != nil {
		return nil, err
	}

	return core.HTMLResponse(html), nil
}

var _ core.Controller = (*PageController)(nil)
