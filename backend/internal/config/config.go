package config

import (
	"net"
	"strconv"
	"time"

	"github.com/vernal96/go-cms/internal/connectors/corecache"
	"github.com/vernal96/go-cms/internal/connectors/corefiles"
	"github.com/vernal96/go-cms/internal/connectors/maineventbus"
	"github.com/vernal96/go-cms/internal/connectors/mainlogger"
	"github.com/vernal96/go-cms/internal/connectors/mainpostgres"
	"github.com/vernal96/go-cms/internal/profiles/dev"
	jwtsecurity "github.com/vernal96/go-cms/internal/security/jwt"
	"github.com/vernal96/go-cms/kernel"
	appkernel "github.com/vernal96/go-cms/kernel/app"
	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	corepostgres "github.com/vernal96/go-cms/kernel/modules/core/adapters/postgres"
)

type Config struct {
	Logger    mainlogger.Config   `envconfig:"LOGGER"`
	EventBus  maineventbus.Config `envconfig:"EVENT_BUS"`
	Server    ServerConfig        `envconfig:"SERVER"`
	Postgres  mainpostgres.Config `envconfig:"POSTGRES"`
	Files     FilesConfig         `envconfig:"FILES"`
	CoreCache corecache.Config    `envconfig:"CORE_CACHE"`
	JWT       jwtsecurity.Config  `envconfig:"JWT"`
}

type FilesConfig struct {
	Public        corefiles.Config `envconfig:"PUBLIC"`
	Private       corefiles.Config `envconfig:"PRIVATE"`
	MaxUploadSize int64            `envconfig:"MAX_UPLOAD_SIZE" default:"104857600"`
	UploadTimeout time.Duration    `envconfig:"UPLOAD_TIMEOUT" default:"10m"`
	AvatarStorage filesystem.Code  `envconfig:"AVATAR_STORAGE" default:"private"`
	AvatarMaxSize int64            `envconfig:"AVATAR_MAX_SIZE" default:"5242880"`
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
		Logger:   mainlogger.NewFactory(c.Logger),
		EventBus: maineventbus.NewFactory(c.EventBus),
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
		Profiles:         []kernel.Profile{dev.Profile},
		SiteAccessPolicy: admin.AllowAllSitesPolicy{},
		MaxUploadSize:    c.Files.MaxUploadSize,
		UploadTimeout:    c.Files.UploadTimeout,
		AvatarStorage:    c.Files.AvatarStorage,
		AvatarMaxSize:    c.Files.AvatarMaxSize,
	}
}
