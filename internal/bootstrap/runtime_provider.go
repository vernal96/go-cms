package bootstrap

import (
	"github.com/vernal96/go-cms/internal/config"
	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/kernel"
	coresite "github.com/vernal96/go-cms/kernel/modules/core/site"
	sitememory "github.com/vernal96/go-cms/kernel/modules/core/site/adapters/memory"
)

func NewRuntimeProvider(
	app *kernel.App,
	profiles kernel.ProfileRegistry,
	cfg *config.Config,
) (*coresite.RuntimeProvider, error) {
	memorySiteRepository := sitememory.NewRepository([]coresite.Site{
		{
			ID:          1,
			ProfileCode: dev.ProfileCode,
			Domain:      cfg.Server.Address(),
			Locale:      "ru-RU",
			Settings:    map[string]any{},
		},
	})

	if err := app.Adapters().Add(
		coresite.RepositoryAdapterContract,
		sitememory.AdapterCode,
		memorySiteRepository,
	); err != nil {
		return nil, err
	}

	siteRepository, err := kernel.AdapterAs[coresite.Repository](
		app.Adapters(),
		coresite.RepositoryAdapterContract,
		sitememory.AdapterCode,
	)
	if err != nil {
		return nil, err
	}

	domainResolver, err := coresite.NewRepositoryDomainResolver(siteRepository)
	if err != nil {
		return nil, err
	}

	runtimeFactory := kernel.NewSiteRuntimeFactory(app)

	return coresite.NewRuntimeProvider(
		domainResolver,
		profiles,
		runtimeFactory,
	)
}
