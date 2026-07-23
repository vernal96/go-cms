package config

import (
	"net"
	"strconv"
	"time"

	"github.com/vernal96/go-cms/internal/connectors/corecache"
	"github.com/vernal96/go-cms/internal/connectors/corefiles"
	"github.com/vernal96/go-cms/internal/connectors/mainpostgres"
	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/kernel"
	appkernel "github.com/vernal96/go-cms/kernel/app"
	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/filesystem"
	corepostgres "github.com/vernal96/go-cms/kernel/modules/core/adapters/postgres"
)

type Config struct {
	Server    ServerConfig        `envconfig:"SERVER"`
	Postgres  mainpostgres.Config `envconfig:"POSTGRES"`
	Files     FilesConfig         `envconfig:"FILES"`
	CoreCache corecache.Config    `envconfig:"CORE_CACHE"`
}

type FilesConfig struct {
	Public  corefiles.Config `envconfig:"PUBLIC"`
	Private corefiles.Config `envconfig:"PRIVATE"`
}

type ServerConfig struct {
	Host            string        `envconfig:"HOST" default:"localhost"`
	Port            int           `envconfig:"PORT" default:"8080"`
	ReadTimeout     time.Duration `envconfig:"READ_TIMEOUT" default:"5s"`
	WriteTimeout    time.Duration `envconfig:"WRITE_TIMEOUT" default:"10s"`
	IdleTimeout     time.Duration `envconfig:"IDLE_TIMEOUT" default:"120s"`
	ShutdownTimeout time.Duration `envconfig:"SHUTDOWN_TIMEOUT" default:"5s"`
}

func (c ServerConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

// Application is a declarative description of this application instance.
func (c Config) Application() appkernel.Definition {
	return appkernel.Definition{
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: mainpostgres.Factory(c.Postgres),
			Adapters: []kernel.ModuleDatabaseFactory{
				corepostgres.DatabaseFactory{},
			},
		},
		Filesystems: []filesystem.Factory{
			corefiles.PublicFactory(c.Files.Public),
			corefiles.PrivateFactory(c.Files.Private),
		},
		Caches: []cache.Factory{
			corecache.NewFactory(c.CoreCache),
		},
		Profiles: []kernel.Profile{dev.Profile},
	}
}
