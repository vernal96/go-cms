package dev

import (
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core"
)

const ProfileCode kernel.ProfileCode = "dev"

var Profile = kernel.Profile{
	Code: ProfileCode,
	Modules: []kernel.ProfileModule{
		{Module: core.Module{}},
	},
}
