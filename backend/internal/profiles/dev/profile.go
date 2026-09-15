package dev

import (
	"time"

	"github.com/vernal96/go-cms/internal/connectors/projectcache"
	devtemplates "github.com/vernal96/go-cms/internal/profiles/dev/templates"
	"github.com/vernal96/go-cms/internal/profiles/dev/widgetviews"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/modules/forms"
	"github.com/vernal96/go-cms/kernel/modules/mail"
	"github.com/vernal96/go-cms/kernel/modules/search"
	"github.com/vernal96/go-cms/kernel/modules/seo"
)

const ProfileCode kernel.ProfileCode = "dev"

func Profile(
	mailConfig mail.Config,
	formsConfig forms.Config,
	spoolStorage filesystem.Code,
) kernel.Profile {
	return kernel.Profile{
		Code: ProfileCode, Name: "Разработка", Params: Params(), EditorTabs: ParamEditorTabs(),
		Templates: devtemplates.All(), WidgetViews: []widget.View{widgetviews.ContentCompact, widgetviews.ContentArticle},
		Modules: []kernel.ProfileModule{
			{Module: core.Module{}, Config: core.Config{RepositoryCacheTTL: 5 * time.Minute}, Caches: []cache.Binding{
				{Alias: core.DurableCacheAlias, Code: projectcache.FilesystemCode},
				{Alias: core.HotCacheAlias, Code: projectcache.RedisCode},
			}},
			{Module: seo.Module{}},
			{Module: mail.Module{}, Config: mailConfig, Filesystems: []filesystem.Binding{{Alias: mail.SpoolFilesystemAlias, Code: spoolStorage}}},
			{Module: forms.Module{}, Config: formsConfig, Filesystems: []filesystem.Binding{{Alias: forms.SpoolFilesystemAlias, Code: spoolStorage}}},
			{Module: search.Module{}},
			{Module: admin.Module{}},
		},
	}
}
