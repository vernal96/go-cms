package dev

import (
	"time"

	"github.com/vernal96/go-cms/internal/connectors/corecache"
	devtemplates "github.com/vernal96/go-cms/internal/profiles/dev/templates"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/modules/seo"
)

const ProfileCode kernel.ProfileCode = "dev"

var Profile = kernel.Profile{
	Code:      ProfileCode,
	Name:      "Разработка",
	Params:    Params(),
	Templates: devtemplates.All(),
	WidgetViews: []widget.ViewDeclaration{
		{Widget: "core_content", Code: "compact", Label: "Компактный"},
		{Widget: "core_content", Code: "article", Label: "Статья"},
	},
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
		{
			Module: seo.Module{},
		},
		{
			Module: admin.Module{},
		},
	},
}
