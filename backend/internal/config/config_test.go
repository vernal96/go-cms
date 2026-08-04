package config_test

import (
	"testing"
	"time"

	projectconfig "github.com/vernal96/go-cms/internal/config"
	"github.com/vernal96/go-cms/internal/connectors/corecache"
	"github.com/vernal96/go-cms/internal/connectors/mainpostgres"
	configloader "github.com/vernal96/go-cms/kernel/config"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	"github.com/vernal96/go-cms/kernel/modules/core"
)

func TestProjectConfigLoadsNestedPrefixesAndBuildsDefinition(t *testing.T) {
	t.Setenv("LOGGER_DRIVER", "file")
	t.Setenv("LOGGER_LEVEL", "info")
	t.Setenv("LOGGER_SERVICE_NAME", "go-cms-test")
	t.Setenv("LOGGER_ENVIRONMENT", "test")
	t.Setenv("EVENT_BUS_DRIVER", "kafka")
	t.Setenv("EVENT_BUS_KAFKA_BROKERS", "kafka-one:9092,kafka-two:9092")
	t.Setenv("EVENT_BUS_KAFKA_CLIENT_ID", "go-cms-test")
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("POSTGRES_HOST", "database")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_DB", "cms")
	t.Setenv("POSTGRES_USER", "cms")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("FILES_PUBLIC_DRIVER", "local")
	t.Setenv("FILES_PUBLIC_LOCAL_ROOT", "/tmp/cms-public")
	t.Setenv("FILES_PRIVATE_DRIVER", "s3")
	t.Setenv("FILES_PRIVATE_S3_REGION", "us-east-1")
	t.Setenv("FILES_PRIVATE_S3_BUCKET", "cms-private")
	t.Setenv("CORE_CACHE_DRIVER", "redis")
	t.Setenv("CORE_CACHE_REDIS_ADDRS", "redis-one:6379,redis-two:6379")
	t.Setenv("CORE_CACHE_REDIS_MASTER_NAME", "cms-primary")
	t.Setenv(
		"JWT_SIGNING_KEY",
		"0123456789abcdef0123456789abcdef",
	)
	t.Setenv("JWT_ISSUER", "cms-test")
	t.Setenv("JWT_AUDIENCE", "cms-test-api")
	t.Setenv("JWT_ACCESS_TTL", "10m")
	t.Setenv("JWT_CLOCK_SKEW", "5s")

	config, err := configloader.Load[projectconfig.Config]("")
	if err != nil {
		t.Fatal(err)
	}
	if config.Server.Address() != "127.0.0.1:9090" {
		t.Fatalf("server address = %q", config.Server.Address())
	}
	if config.EventBus.Driver != "kafka" ||
		len(config.EventBus.Kafka.Brokers) != 2 ||
		config.EventBus.Kafka.Brokers[0] != "kafka-one:9092" ||
		config.EventBus.Kafka.ClientID != "go-cms-test" {
		t.Fatalf("event bus config = %#v", config.EventBus)
	}
	if config.Postgres.Host != "database" || config.Postgres.Database != "cms" {
		t.Fatalf("postgres config = %#v", config.Postgres)
	}
	if config.CoreCache.Driver != "redis" ||
		len(config.CoreCache.Redis.Addrs) != 2 ||
		config.CoreCache.Redis.MasterName != "cms-primary" {
		t.Fatalf("core cache config = %#v", config.CoreCache)
	}
	if config.JWT.SigningKey !=
		"0123456789abcdef0123456789abcdef" ||
		config.JWT.Issuer != "cms-test" ||
		config.JWT.Audience != "cms-test-api" ||
		config.JWT.AccessTTL != 10*time.Minute ||
		config.JWT.ClockSkew != 5*time.Second {
		t.Fatal("JWT configuration was not loaded correctly")
	}

	definition := config.Application()
	if definition.Logger == nil {
		t.Fatal("logger factory is nil")
	}
	if definition.EventBus == nil {
		t.Fatal("event bus factory is nil")
	}
	if definition.MainDatabase.Connector.Code() != mainpostgres.ConnectionCode {
		t.Fatalf("connection code = %q", definition.MainDatabase.Connector.Code())
	}
	if len(definition.MainDatabase.Adapters) != 1 ||
		definition.MainDatabase.Adapters[0].ModuleCode() != core.ModuleCode {
		t.Fatalf("database adapters = %#v", definition.MainDatabase.Adapters)
	}
	if len(definition.Profiles) != 1 || definition.Profiles[0].Code != "dev" {
		t.Fatalf("profiles = %#v", definition.Profiles)
	}
	if len(definition.Filesystems) != 2 ||
		definition.Filesystems[0].Code() != filesystem.Code("public") ||
		definition.Filesystems[1].Code() != filesystem.Code("private") {
		t.Fatalf("filesystem factories = %#v", definition.Filesystems)
	}
	if len(definition.Caches) != 1 ||
		definition.Caches[0].Code() != corecache.Code {
		t.Fatalf("cache factories = %#v", definition.Caches)
	}
	if len(definition.Profiles[0].Modules) != 2 ||
		definition.Profiles[0].Modules[0].Module.Code() != core.ModuleCode ||
		definition.Profiles[0].Modules[1].Module.Code() != admin.ModuleCode ||
		len(definition.Profiles[0].Modules[0].Caches) != 1 ||
		definition.Profiles[0].Modules[0].Caches[0].Code != corecache.Code {
		t.Fatalf(
			"core cache bindings = %#v",
			definition.Profiles[0].Modules,
		)
	}
}
