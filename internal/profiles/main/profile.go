package main

import (
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core"
)

type Profile struct{}

func (p Profile) Code() kernel.ProfileCode {
	return "main"
}

func (p Profile) Modules() []kernel.Module {
	return []kernel.Module{
		core.Module{},
	}
}

var _ kernel.Profile = Profile{}
