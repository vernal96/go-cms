package configs

import (
	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/internal/project"
)

func Null() project.Config {
	return project.Config{
		CacheStores: nil,
		FileDisks:   nil,
		Events:      core.NullEventBus{},
	}
}
