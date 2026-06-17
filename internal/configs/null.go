package configs

import (
	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/project"
)

func NullInfrastructure() project.InfrastructureConfig {
	return project.InfrastructureConfig{
		CacheStores: nil,
		FileDisks:   nil,
		Events:      core.NullEventBus{},
	}
}

func NullSiteProfiles() project.SiteProfileConfig {
	return project.SiteProfileConfig{
		Profiles: nil,
	}
}
