package config_test

import (
	"testing"
	"time"

	projectconfig "github.com/vernal96/go-cms/internal/config"
	"github.com/vernal96/go-cms/internal/connectors/mainpostgres"
	"github.com/vernal96/go-cms/internal/connectors/projectcache"
	configloader "github.com/vernal96/go-cms/kernel/config"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/mail"
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
	t.Setenv("FILES_MAX_UPLOAD_SIZE", "209715200")
	t.Setenv("FILES_UPLOAD_TIMEOUT", "12m")
	t.Setenv("FILES_AVATAR_STORAGE", "private")
	t.Setenv("FILES_AVATAR_MAX_SIZE", "4194304")
	t.Setenv("CACHE_FILESYSTEM_STORAGE", "private")
	t.Setenv("CACHE_REDIS_ADDRS", "redis-one:6379,redis-two:6379")
	t.Setenv("CACHE_REDIS_MASTER_NAME", "cms-primary")
	t.Setenv(
		"JWT_SIGNING_KEY",
		"0123456789abcdef0123456789abcdef",
	)
	t.Setenv("JWT_ISSUER", "cms-test")
	t.Setenv("JWT_AUDIENCE", "cms-test-api")
	t.Setenv("JWT_ACCESS_TTL", "10m")
	t.Setenv("JWT_CLOCK_SKEW", "5s")
	t.Setenv("OUTBOX_BATCH_SIZE", "25")
	t.Setenv("OUTBOX_CLEANUP_MAX_BATCHES", "40")
	t.Setenv("MAIL_TRANSPORT_DEFAULT_DRIVER", "null")
	t.Setenv("MAIL_HISTORY_RETENTION", "240h")

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
	if config.Caches.Filesystem.Storage != "private" ||
		len(config.Caches.Redis.Addrs) != 2 ||
		config.Caches.Redis.MasterName != "cms-primary" {
		t.Fatalf("cache config = %#v", config.Caches)
	}
	if config.Files.MaxUploadSize != 209715200 || config.Files.UploadTimeout != 12*time.Minute ||
		config.Files.AvatarStorage != "private" || config.Files.AvatarMaxSize != 4194304 {
		t.Fatalf("file upload config = %#v", config.Files)
	}
	if config.JWT.SigningKey !=
		"0123456789abcdef0123456789abcdef" ||
		config.JWT.Issuer != "cms-test" ||
		config.JWT.Audience != "cms-test-api" ||
		config.JWT.AccessTTL != 10*time.Minute ||
		config.JWT.ClockSkew != 5*time.Second {
		t.Fatal("JWT configuration was not loaded correctly")
	}
	if config.Outbox.BatchSize != 25 || config.Outbox.PollInterval != 500*time.Millisecond ||
		config.Outbox.LeaseDuration != 30*time.Second || config.Outbox.PublishedRetention != 7*24*time.Hour ||
		config.Outbox.CleanupMaxBatches != 40 {
		t.Fatalf("outbox configuration = %#v", config.Outbox)
	}
	if config.Mail.DefaultDriver != "null" || config.Mail.HistoryRetention != 240*time.Hour {
		t.Fatalf("mail configuration = %#v", config.Mail)
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
	if len(definition.MainDatabase.Adapters) != 3 ||
		definition.MainDatabase.Adapters[0].ModuleCode() != core.ModuleCode ||
		definition.MainDatabase.Adapters[1].ModuleCode() != "seo" ||
		definition.MainDatabase.Adapters[2].ModuleCode() != mail.ModuleCode {
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
	if len(definition.Caches) != 2 ||
		definition.Caches[0].Code() != projectcache.FilesystemCode ||
		definition.Caches[1].Code() != projectcache.RedisCode {
		t.Fatalf("cache factories = %#v", definition.Caches)
	}
	if definition.OutboxPublisher.BatchSize != 25 || definition.OutboxPublisher.CleanupInterval != time.Hour ||
		definition.OutboxPublisher.CleanupMaxBatches != 40 {
		t.Fatalf("outbox publisher definition = %#v", definition.OutboxPublisher)
	}
	if len(definition.Profiles[0].Modules) != 4 ||
		definition.Profiles[0].Modules[0].Module.Code() != core.ModuleCode ||
		definition.Profiles[0].Modules[1].Module.Code() != "seo" ||
		definition.Profiles[0].Modules[2].Module.Code() != mail.ModuleCode ||
		definition.Profiles[0].Modules[3].Module.Code() != admin.ModuleCode ||
		len(definition.Profiles[0].Modules[0].Caches) != 2 ||
		definition.Profiles[0].Modules[0].Caches[0].Alias != core.DurableCacheAlias ||
		definition.Profiles[0].Modules[0].Caches[0].Code != projectcache.FilesystemCode ||
		definition.Profiles[0].Modules[0].Caches[1].Alias != core.HotCacheAlias ||
		definition.Profiles[0].Modules[0].Caches[1].Code != projectcache.RedisCode {
		t.Fatalf(
			"core cache bindings = %#v",
			definition.Profiles[0].Modules,
		)
	}
}
