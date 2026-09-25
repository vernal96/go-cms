package settings

import (
	"io/fs"
	"time"

	kernel "github.com/vernal96/go-cms-kernel"
	appkernel "github.com/vernal96/go-cms-kernel/app"
	"github.com/vernal96/go-cms-kernel/modules/core"
	"github.com/vernal96/go-cms-kernel/seeds"

	"github.com/vernal96/go-cms/internal/platform"
)

// Config gathers the project-level choices that complete an application definition.
type Config struct {
	LoggerPath     string
	Infrastructure appkernel.Definition
	Profile        kernel.Profile
	DevSeed        bool
	SeedFiles      fs.FS
}

// Definition combines infrastructure, profile, and project settings for app.New.
func (c Config) Definition() appkernel.Definition {
	definition := c.Infrastructure
	definition.Logger = platform.LoggerFactory{Path: c.LoggerPath}
	definition.Profiles = []kernel.Profile{c.Profile}
	definition.AvatarStorage = "private"
	definition.MaxUploadSize = 100 << 20
	definition.UploadTimeout = 10 * time.Minute
	definition.AvatarMaxSize = 5 << 20
	if c.DevSeed {
		definition.MainDatabase.Seeds = []appkernel.ModuleSeedSource{{
			Module: core.ModuleCode,
			Source: seeds.Source{ID: "starter", Schema: "core", Tags: []seeds.Tag{"dev"}, FS: c.SeedFiles, Path: "seeds"},
		}}
	}
	return definition
}
