package main

import (
	"github.com/vernal96/go-cms/core"
	coremodule "github.com/vernal96/go-cms/core/modules/core"
)

const Code = "main"

type Profile struct{}

func New() *Profile {
	return &Profile{}
}

func (p *Profile) Code() string {
	return Code
}

func (p *Profile) Modules() []core.Module {
	return []core.Module{
		coremodule.New(coremodule.Config{}),
	}
}

var _ core.SiteProfile = (*Profile)(nil)
