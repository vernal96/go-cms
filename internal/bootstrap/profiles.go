package bootstrap

import (
	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/kernel"
)

func projectProfiles() []kernel.Profile {
	return []kernel.Profile{
		dev.Profile{},
	}
}
