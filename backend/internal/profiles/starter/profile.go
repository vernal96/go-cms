package starter

import (
	"time"

	kernel "github.com/vernal96/go-cms-kernel"
	"github.com/vernal96/go-cms-kernel/cache"
	"github.com/vernal96/go-cms-kernel/modules/admin"
	"github.com/vernal96/go-cms-kernel/modules/core"
	"github.com/vernal96/go-cms-kernel/modules/core/resourceviews"
)

// Profile is the profile and module bindings used by the starter app.
var Profile = kernel.Profile{
	Code: "starter",
	Name: "Starter",
	Modules: []kernel.Module{
		core.New(core.Config{
			Caches: []cache.Binding{
				{Alias: core.DurableCacheAlias, Code: "shared"},
				{Alias: core.HotCacheAlias, Code: "shared"},
			},
			ResourceViews: &resourceviews.Config{Accumulator: "views", FlushInterval: 45 * time.Second},
		}),
		admin.New(),
	},
}
