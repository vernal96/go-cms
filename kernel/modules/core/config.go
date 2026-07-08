package core

import "github.com/vernal96/go-cms/kernel"

type Config struct {
	AdapterDefaults kernel.AdapterDefaults
	Site            SiteConfig
}

type SiteConfig struct {
	RepositoryAdapter kernel.AdapterDefaults
}
