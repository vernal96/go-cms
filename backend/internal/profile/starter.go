package profile

import (
	kernel "github.com/vernal96/go-cms-kernel"
	"github.com/vernal96/go-cms-kernel/cache"
	"github.com/vernal96/go-cms-kernel/modules/admin"
	"github.com/vernal96/go-cms-kernel/modules/core"
)

// Starter is the profile and module bindings used by the starter app.
var Starter = kernel.Profile{Code: "starter", Name: "Starter", Modules: []kernel.ProfileModule{
	{Module: core.Module{}, Caches: []cache.Binding{
		{Alias: core.DurableCacheAlias, Code: "shared"},
		{Alias: core.HotCacheAlias, Code: "shared"},
	}},
	{Module: admin.Module{}},
}}
