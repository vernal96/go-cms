package bootstrap

import (
	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/kernel"
)

func NewProfileRegistry() (*kernel.ProfileRegistryManager, error) {
	profiles := kernel.NewProfileRegistryManager()

	if err := profiles.Register(dev.Profile{}); err != nil {
		return nil, err
	}

	return profiles, nil
}
