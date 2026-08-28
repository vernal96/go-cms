package dev

import (
	"time"

	"github.com/vernal96/go-cms/internal/connectors/corefiles"
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
	"github.com/vernal96/go-cms/kernel/modules/seo"
)

const ProfileCode kernel.ProfileCode = "dev"

var Profile = kernel.Profile{
	Code:       ProfileCode,
	Name:       "Разработка",
	Params:     Params(),
	EditorTabs: ParamEditorTabs(),
	Templates:  devtemplates.All(),
	WidgetViews: []widget.View{
		widgetviews.ContentCompact,
		widgetviews.ContentArticle,
	},
	Modules: []kernel.ProfileModule{
		{
			Module: core.Module{},
			Config: core.Config{
				RepositoryCacheTTL: 5 * time.Minute,
			},
			Caches: []cache.Binding{
				{
					Alias: core.DurableCacheAlias,
					Code:  projectcache.FilesystemCode,
				},
				{
					Alias: core.HotCacheAlias,
					Code:  projectcache.RedisCode,
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

func ProfileWithMailAndForms(mailConfig mail.Config, formsConfig forms.Config) kernel.Profile {
	result := Profile
	result.Modules = append([]kernel.ProfileModule(nil), Profile.Modules...)
	mailModule := kernel.ProfileModule{Module: mail.Module{}, Config: mailConfig, Filesystems: []filesystem.Binding{{Alias: mail.SpoolFilesystemAlias, Code: corefiles.PrivateCode}}}
	formsModule := kernel.ProfileModule{Module: forms.Module{}, Config: formsConfig, Filesystems: []filesystem.Binding{{Alias: forms.SpoolFilesystemAlias, Code: corefiles.PrivateCode}}}
	adminIndex := len(result.Modules) - 1
	result.Modules = append(result.Modules, kernel.ProfileModule{}, kernel.ProfileModule{})
	copy(result.Modules[adminIndex+2:], result.Modules[adminIndex:])
	result.Modules[adminIndex] = mailModule
	result.Modules[adminIndex+1] = formsModule
	return result
}
