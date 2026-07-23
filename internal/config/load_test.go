package config_test

import (
	"os"
	"path/filepath"
	"testing"

	projectconfig "github.com/vernal96/go-cms/internal/config"
)

func TestLoadReadsDotEnvAndPreservesProcessEnvironment(t *testing.T) {
	t.Chdir(t.TempDir())

	dotEnv := "POSTGRES_HOST=file-host\r\n" +
		"POSTGRES_PORT=5432\r\n" +
		"POSTGRES_DB=cms\r\n" +
		"POSTGRES_USER=cms\r\n" +
		"POSTGRES_PASSWORD=secret\r\n"
	if err := os.WriteFile(filepath.Join(".", ".env"), []byte(dotEnv), 0o600); err != nil {
		t.Fatal(err)
	}

	keys := []string{
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_DB",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
	}
	for _, key := range keys {
		value, exists := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if exists {
				_ = os.Setenv(key, value)
				return
			}
			_ = os.Unsetenv(key)
		})
	}

	if err := os.Setenv("POSTGRES_HOST", "environment-host"); err != nil {
		t.Fatal(err)
	}

	config, err := projectconfig.Load()
	if err != nil {
		t.Fatal(err)
	}
	if config.Postgres.Host != "environment-host" {
		t.Fatalf("postgres host = %q", config.Postgres.Host)
	}
	if config.Postgres.Port != 5432 || config.Postgres.Database != "cms" {
		t.Fatalf("postgres config = %#v", config.Postgres)
	}
}
