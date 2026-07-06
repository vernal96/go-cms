package bootstrap

import (
	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/kernel"
)

var projectProfiles = []kernel.Profile{
	dev.Profile{},
}

func NewProfileRegistry() (*kernel.ProfileRegistryManager, error) {
	profiles := kernel.NewProfileRegistryManager()

	for _, profile := range projectProfiles {
		if err := profiles.Register(profile); err != nil {
			return nil, err
		}
	}

	return profiles, nil
}
