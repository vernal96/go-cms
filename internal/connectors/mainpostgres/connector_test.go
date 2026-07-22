package mainpostgres_test

import (
	"testing"

	"github.com/vernal96/go-cms/internal/connectors/logspostgres"
	"github.com/vernal96/go-cms/internal/connectors/mainpostgres"
)

func TestConnectionConfigsUseIndependentEnvironmentNames(t *testing.T) {
	t.Setenv("POSTGRES_HOST", "main-host")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_DB", "main-db")
	t.Setenv("POSTGRES_USER", "main-user")
	t.Setenv("POSTGRES_PASSWORD", "main-password")
	t.Setenv("POSTGRES_MIN_CONNS", "2")

	t.Setenv("LOGS_POSTGRES_HOST", "logs-host")
	t.Setenv("LOGS_POSTGRES_PORT", "6432")
	t.Setenv("LOGS_POSTGRES_DB", "logs-db")
	t.Setenv("LOGS_POSTGRES_USER", "logs-user")
	t.Setenv("LOGS_POSTGRES_PASSWORD", "logs-password")
	t.Setenv("LOGS_POSTGRES_MIN_CONNS", "3")

	mainConfig, err := mainpostgres.Load()
	if err != nil {
		t.Fatal(err)
	}
	logsConfig, err := logspostgres.Load()
	if err != nil {
		t.Fatal(err)
	}

	if mainConfig.Host != "main-host" || mainConfig.Database != "main-db" {
		t.Fatalf("unexpected main config: %#v", mainConfig)
	}
	if logsConfig.Host != "logs-host" || logsConfig.Database != "logs-db" {
		t.Fatalf("unexpected logs config: %#v", logsConfig)
	}
	if mainConfig.MinConns != 2 || logsConfig.MinConns != 3 {
		t.Fatalf(
			"min connections = main:%d logs:%d",
			mainConfig.MinConns,
			logsConfig.MinConns,
		)
	}
	if mainpostgres.ConnectionCode == logspostgres.ConnectionCode {
		t.Fatal("main and logs connections have the same code")
	}
}
