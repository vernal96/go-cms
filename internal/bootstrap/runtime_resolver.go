package bootstrap

import (
	"github.com/vernal96/go-cms/internal/config"
	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/kernel"
	coresite "github.com/vernal96/go-cms/kernel/modules/core/site"
	sitememory "github.com/vernal96/go-cms/kernel/modules/core/site/adapters/memory"
)

func NewRuntimeResolver(
	app *kernel.App,
	profiles kernel.ProfileRegistry,
	cfg *config.Config,
) (*coresite.RuntimeResolver, error) {
	siteRepository := sitememory.NewRepository([]coresite.Site{
		{
			ID:          1,
			ProfileCode: dev.ProfileCode,
			Domain:      cfg.Server.Address(),
			Locale:      "ru-RU",
			Settings:    map[string]any{},
		},
	})

	siteResolver, err := coresite.NewRepositoryResolver(siteRepository)
	if err != nil {
		return nil, err
	}

	runtimeFactory := kernel.NewSiteRuntimeFactory(app)

	return coresite.NewRuntimeResolver(
		siteResolver,
		profiles,
		runtimeFactory,
	)
}
