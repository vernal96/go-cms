package dev

import (
	"time"

	"github.com/vernal96/go-cms/internal/connectors/corecache"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/modules/core"
)

const ProfileCode kernel.ProfileCode = "dev"

var Profile = kernel.Profile{
	Code: ProfileCode,
	Modules: []kernel.ProfileModule{
		{
			Module: core.Module{},
			Config: core.Config{
				RepositoryCacheTTL: 5 * time.Minute,
			},
			Caches: []cache.Binding{
				{
					Alias:     core.RepositoryCacheAlias,
					Code:      corecache.Code,
					Namespace: "core/repository",
				},
			},
		},
	},
}
