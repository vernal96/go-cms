package infrastructure

import (
	"time"

	kernel "github.com/vernal96/go-cms-kernel"
	appkernel "github.com/vernal96/go-cms-kernel/app"
	"github.com/vernal96/go-cms-kernel/cache"
	"github.com/vernal96/go-cms-kernel/connectors/kafkaeventbus"
	"github.com/vernal96/go-cms-kernel/connectors/localstorage"
	connectorpostgres "github.com/vernal96/go-cms-kernel/connectors/postgres"
	connectorredis "github.com/vernal96/go-cms-kernel/connectors/redis"
	"github.com/vernal96/go-cms-kernel/filesystem"
	corepostgres "github.com/vernal96/go-cms-kernel/modules/core/adapters/postgres"
	"github.com/vernal96/go-cms-kernel/modules/core/user/adapters/argon2id"
)

// Config contains deployment-specific settings for the starter connectors.
type Config struct {
	KafkaBrokers        []string
	PostgresHost        string
	PostgresPort        int
	PostgresDatabase    string
	PostgresUser        string
	PostgresPassword    string
	PostgresSSLMode     string
	PublicFilesRoot     string
	PublicFilesBaseURL  string
	PrivateFilesRoot    string
	PrivateFilesBaseURL string
	PrivateFilesSignKey string
	RedisAddress        string
	RedisPassword       string
}

// Definition returns the infrastructure declarations used by the starter app.
func (c Config) Definition() appkernel.Definition {
	return appkernel.Definition{
		EventBus: kafkaeventbus.Factory{Config: kafkaeventbus.Config{
			Brokers: c.KafkaBrokers, ClientID: "go-cms", DialTimeout: 5 * time.Second,
			ConsumerRetryDelay: time.Second, ShutdownTimeout: 5 * time.Second,
		}},
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: connectorpostgres.Factory{Config: connectorpostgres.Config{
				Code: "main", Host: c.PostgresHost, Port: c.PostgresPort, Database: c.PostgresDatabase,
				User: c.PostgresUser, Password: c.PostgresPassword, SSLMode: c.PostgresSSLMode,
				MaxConns: 10, ConnMaxLifetime: time.Hour, ConnectTimeout: 5 * time.Second,
			}},
			Adapters: []kernel.ModuleDatabaseFactory{corepostgres.DatabaseFactory{}},
		},
		Filesystems: []filesystem.Factory{
			localstorage.Factory{Config: localstorage.Config{
				Code: "public", Visibility: filesystem.VisibilityPublic, Root: c.PublicFilesRoot, BaseURL: c.PublicFilesBaseURL,
			}},
			localstorage.Factory{Config: localstorage.Config{
				Code: "private", Visibility: filesystem.VisibilityPrivate, Root: c.PrivateFilesRoot,
				BaseURL: c.PrivateFilesBaseURL, SigningKey: c.PrivateFilesSignKey,
			}},
		},
		Caches: []cache.Factory{connectorredis.Factory{Config: connectorredis.Config{
			Code: "shared", Addrs: []string{c.RedisAddress}, Password: c.RedisPassword,
			DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second,
		}}},
		PasswordHasher: argon2id.Factory{},
	}
}
