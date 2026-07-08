package core

import "github.com/vernal96/go-cms/kernel"

type Config struct {
	AdapterDefaults kernel.AdapterDefaults
	Site            SiteConfig
}

type SiteConfig struct {
	AdapterDefaults   kernel.AdapterDefaults
	RepositoryAdapter kernel.AdapterCode `cms:"adapter,contract=core.site.repository,default=repository"`
}
