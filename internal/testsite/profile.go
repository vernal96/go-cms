package testsite

import (
	"95.79.129.33/go-cms/dev/core"
	"95.79.129.33/go-cms/dev/internal/testmodule"
)

type Profile struct{}

func New() *Profile {
	return &Profile{}
}

func (p *Profile) Code() string {
	return "main"
}

func (p *Profile) Modules() []core.Module {
	return []core.Module{
		testmodule.New(),
	}
}
