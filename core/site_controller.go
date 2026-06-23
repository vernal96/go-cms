package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

const (
	SiteInfoRoutePath       = "/_cms/site"
	SiteInfoWidgetFullCode  = "core.site_info"
)

type SiteController struct{}

func NewSiteController() *SiteController {
	return &SiteController{}
}

func (c *SiteController) Routes() []Route {
	return []Route{
		{
			Method:  RouteMethodGet,
			Path:    SiteInfoRoutePath,
			Handler: c.siteInfo,
		},
	}
}

func (c *SiteController) siteInfo(
	ctx context.Context,
	runtime *SiteRuntime,
	request *http.Request,
) (any, error) {
	if runtime == nil {
		return nil, errors.New("site runtime is nil")
	}

	widget, exists := runtime.Registry().Widgets().Get(SiteInfoWidgetFullCode)
	if !exists {
		return nil, fmt.Errorf("widget %q is not registered", SiteInfoWidgetFullCode)
	}

	return widget.Render(ctx, WidgetParams{
		string(SiteInfoWidgetParamSite): runtime.Site(),
	})
}

var _ Controller = (*SiteController)(nil)
