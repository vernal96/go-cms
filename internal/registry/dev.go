package registry

import (
	"github.com/vernal96/go-cms/adapters/cache/memorycache"
	"github.com/vernal96/go-cms/adapters/eventbus/memoryeventbus"
	"github.com/vernal96/go-cms/adapters/storage/memorystorage"
	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/project"
	"github.com/vernal96/go-cms/internal/testmodule"
	"github.com/vernal96/go-cms/internal/testsite"
)

func RegisterDevInfrastructure(r *project.InfrastructureRegistry) {
	cacheStore := memorycache.NewStore()

	r.RegisterCacheStore(memorycache.StoreName, cacheStore)

	r.RegisterCacheScope(core.CacheScopeDefault, cacheStore)
	r.RegisterCacheScope(testmodule.CacheScopeDefault, cacheStore)

	r.RegisterFileDisk(
		testmodule.FileDiskDefault,
		memorystorage.NewStorage(),
	)

	r.UseEventBus(memoryeventbus.NewBus())
}

func RegisterDevSiteProfiles(r *project.SiteProfileRegistry) {
	r.Register(testsite.New())
}
