package core

import (
	"context"
	"errors"
)

const (
	SiteInfoWidgetCode WidgetCode = "site_info"
	SiteInfoWidgetParamSite WidgetParamCode = "site"
)

type SiteInfoWidget struct{}

func NewSiteInfoWidget() *SiteInfoWidget {
	return &SiteInfoWidget{}
}

func (w *SiteInfoWidget) Code() WidgetCode {
	return SiteInfoWidgetCode
}

func (w *SiteInfoWidget) Name() string {
	return "Site info"
}

func (w *SiteInfoWidget) Params() []WidgetParamDefinition {
	return nil
}

func (w *SiteInfoWidget) Render(ctx context.Context, params WidgetParams) (WidgetResult, error) {
	if err := ctx.Err(); err != nil {
		return WidgetResult{}, err
	}

	siteValue, exists := params[string(SiteInfoWidgetParamSite)]
	if !exists {
		return WidgetResult{}, errors.New("site info widget site param is missing")
	}

	site, ok := siteValue.(Site)
	if !ok {
		return WidgetResult{}, errors.New("site info widget site param must be core.Site")
	}

	return WidgetResult{
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

var _ Widget = (*SiteInfoWidget)(nil)
