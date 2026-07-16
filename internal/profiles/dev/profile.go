package dev

import (
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core"
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
		},
	}
}

var _ kernel.Profile = Profile{}
