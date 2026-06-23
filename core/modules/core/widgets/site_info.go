package widgets

import (
	"context"

	"github.com/vernal96/go-cms/core"
)

const SiteInfoWidgetCode core.WidgetCode = "site_info"

type SiteInfoWidget struct{}

func NewSiteInfoWidget() *SiteInfoWidget {
	return &SiteInfoWidget{}
}

func (s *SiteInfoWidget) Code() core.WidgetCode {
	return SiteInfoWidgetCode
}

func (s *SiteInfoWidget) Name() string {
	return "Site info"
}

func (s *SiteInfoWidget) Params() []core.WidgetParamDefinition {
	return nil
}

func (s *SiteInfoWidget) Render(ctx context.Context, params core.WidgetParams) (core.WidgetResult, error) {
	if err := ctx.Err(); err != nil {
		return core.WidgetResult{}, err
	}

	return core.WidgetResult{
		Data: map[string]any{
			"title":       "Core module widget",
			"site_id":     params["site_is"],
			"site_domain": params["site_domain"],
			"locale":      params["locale"],
		},
	}, nil
}

var _ core.Widget = (*SiteInfoWidget)(nil)
