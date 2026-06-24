package testsite

import (
	"github.com/vernal96/go-cms/core"
	coremodule "github.com/vernal96/go-cms/core/modules/core"
	"github.com/vernal96/go-cms/internal/testmodule"
)

const FileDiskTest core.FileDisk = "test"

type Profile struct{}

func New() *Profile {
	return &Profile{}
}

func (p *Profile) Code() string {
	return "main"
}

func (p *Profile) Modules() []core.Module {
	return []core.Module{
		coremodule.New(coremodule.Config{}),
		testmodule.New(testmodule.Config{
			CacheScope: testmodule.CacheScopeDefault,
			FileDisk:   FileDiskTest,
		}),
	}
}
