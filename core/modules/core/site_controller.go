package coremodule

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/vernal96/go-cms/core"
)

const (
	SiteInfoRoutePath      = "/_cms/site"
	SiteInfoWidgetFullCode = core.WidgetCode(ModuleCode + ".site_info")
)

type SiteController struct{}

func NewSiteController() *SiteController {
	return &SiteController{}
}

func (c *SiteController) Routes() []core.Route {
	return []core.Route{
		{
			Method:  core.RouteMethodGet,
			Path:    SiteInfoRoutePath,
			Handler: c.siteInfo,
		},
	}
}

func (c *SiteController) siteInfo(
	ctx context.Context,
	runtime *core.SiteRuntime,
	request *http.Request,
) (any, error) {
	if runtime == nil {
		return nil, errors.New("site runtime is nil")
	}

	widget, exists := runtime.Registry().Widgets().Get(SiteInfoWidgetFullCode)
	if !exists {
		return nil, fmt.Errorf("widget %q is not registered", SiteInfoWidgetFullCode)
	}

	return widget.Render(ctx, core.WidgetParams{
		string(SiteInfoWidgetParamSite): runtime.Site(),
	})
}

var _ core.Controller = (*SiteController)(nil)
