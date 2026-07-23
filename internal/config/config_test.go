package config_test

import (
	"testing"

	projectconfig "github.com/vernal96/go-cms/internal/config"
	"github.com/vernal96/go-cms/internal/connectors/mainpostgres"
	configloader "github.com/vernal96/go-cms/kernel/config"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/core"
)

func TestProjectConfigLoadsNestedPrefixesAndBuildsDefinition(t *testing.T) {
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

	config, err := configloader.Load[projectconfig.Config]("")
	if err != nil {
		t.Fatal(err)
	}
	if config.Server.Address() != "127.0.0.1:9090" {
		t.Fatalf("server address = %q", config.Server.Address())
	}
	if config.Postgres.Host != "database" || config.Postgres.Database != "cms" {
		t.Fatalf("postgres config = %#v", config.Postgres)
	}

	definition := config.Application()
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
}
