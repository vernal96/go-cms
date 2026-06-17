package testsite

import (
	"github.com/vernal96/go-cms/adapters/cache/memorycache"
	"github.com/vernal96/go-cms/adapters/storage/memorystorage"
	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/testmodule"
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
		testmodule.New(testmodule.Config{
			CacheStore: memorycache.StoreName,
			FileDisk:   memorystorage.DiskName,
		}),
	}
}
