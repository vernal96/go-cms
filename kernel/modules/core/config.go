package core

import "github.com/vernal96/go-cms/kernel"

type Config struct {
	AdapterDefaults kernel.AdapterDefaults
	Site            SiteConfig
}

type SiteConfig struct {
	RepositoryAdapter kernel.AdapterCode
}

func (c Config) SiteAdapterDefaults(parent kernel.AdapterDefaults) kernel.AdapterDefaults {
	return kernel.ResolveAdapterDefaults(
		parent,
		c.AdapterDefaults,
		kernel.AdapterDefaults{
			RepositoryAdapter: c.Site.RepositoryAdapter,
		},
	)
}
