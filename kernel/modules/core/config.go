package core

import "github.com/vernal96/go-cms/kernel"

type Config struct {
	Site SiteConfig
}

type SiteConfig struct {
	RepositoryAdapter kernel.AdapterCode `default:"postgres"`
}
