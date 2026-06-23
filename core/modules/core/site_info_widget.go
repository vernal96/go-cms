package coremodule

import (
	"context"
	"errors"

	"github.com/vernal96/go-cms/core"
)

const (
	SiteInfoWidgetCode      core.WidgetCode = "site_info"
	SiteInfoWidgetParamSite core.WidgetParamCode = "site"
)

type SiteInfoWidget struct{}

func NewSiteInfoWidget() *SiteInfoWidget {
	return &SiteInfoWidget{}
}

func (w *SiteInfoWidget) Code() core.WidgetCode {
	return SiteInfoWidgetCode
}

func (w *SiteInfoWidget) Name() string {
	return "Site info"
}

func (w *SiteInfoWidget) Params() []core.WidgetParamDefinition {
	return nil
}

func (w *SiteInfoWidget) Render(ctx context.Context, params core.WidgetParams) (core.WidgetResult, error) {
	if err := ctx.Err(); err != nil {
		return core.WidgetResult{}, err
	}

	siteValue, exists := params[string(SiteInfoWidgetParamSite)]
	if !exists {
		return core.WidgetResult{}, errors.New("site info widget site param is missing")
	}

	site, ok := siteValue.(core.Site)
	if !ok {
		return core.WidgetResult{}, errors.New("site info widget site param must be core.Site")
	}

	return core.WidgetResult{
		Data: map[string]any{
			"site": map[string]any{
				"id":           site.ID,
				"profile_code": site.ProfileCode,
				"domain":       site.Domain,
				"locale":       site.Locale,
				"settings":     site.Settings,
			},
		},
	}, nil
}

var _ core.Widget = (*SiteInfoWidget)(nil)
