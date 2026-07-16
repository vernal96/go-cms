package main_profile

import (
	"github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core"
)

const ProfileCode kernel.ProfileCode = "main"

type Profile struct {
}

func (p Profile) Code() kernel.ProfileCode {
	return ProfileCode
}

func (p Profile) Modules() []kernel.Module {
	return []kernel.Module{
		core.Module{
			Config: core.Config{
				DBConnector: &postgres.Connector{},
			},
		},
	}
}

var _ kernel.Profile = Profile{}
