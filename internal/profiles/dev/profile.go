package dev

import (
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core"
	sitememory "github.com/vernal96/go-cms/kernel/modules/core/site/adapters/memory"
)

const ProfileCode kernel.ProfileCode = "dev"

type Profile struct{}

func (p Profile) Code() kernel.ProfileCode {
	return ProfileCode
}

func (p Profile) Modules() []kernel.ProfileModule {
	return []kernel.ProfileModule{
		{
			Module: core.Module{},
			ModuleConfig: core.Config{
				Site: core.SiteConfig{
					RepositoryAdapter: sitememory.AdapterCode,
				},
			},
		},
	}
}

var _ kernel.Profile = Profile{}
