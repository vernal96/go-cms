package postgres

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"testing/fstest"
	"time"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/seeds"
)

func TestPostgresMigrationsAndSiteRepository(t *testing.T) {
	host := os.Getenv("CMS_TEST_POSTGRES_HOST")
	if host == "" {
		t.Skip("set CMS_TEST_POSTGRES_HOST to run the PostgreSQL integration test")
	}

	port := 5432
	if value := os.Getenv("CMS_TEST_POSTGRES_PORT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			t.Fatalf("parse CMS_TEST_POSTGRES_PORT: %v", err)
		}
		port = parsed
	}

	sslMode := os.Getenv("CMS_TEST_POSTGRES_SSL_MODE")
	if sslMode == "" {
		sslMode = "disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	connector, err := connectorpostgres.New(ctx, connectorpostgres.Config{
		Code:            kernel.ConnectionCode("integration"),
		Host:            host,
		Port:            port,
		Database:        os.Getenv("CMS_TEST_POSTGRES_DB"),
		User:            os.Getenv("CMS_TEST_POSTGRES_USER"),
		Password:        os.Getenv("CMS_TEST_POSTGRES_PASSWORD"),
		SSLMode:         sslMode,
		MaxConns:        4,
		MinConns:        0,
		ConnMaxLifetime: time.Minute,
		ConnectTimeout:  5 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connector.Close() })

	if err := connector.Ping(ctx); err != nil {
		t.Fatal(err)
	}

	database, err := NewDatabase(connector)
	if err != nil {
		t.Fatal(err)
	}

	plan := migrations.Plan{
		Connection: string(connector.Code()),
		Target:     connector,
		Source:     database.MigrationSources()[0],
	}
	manager := migrations.NewManager()

	restoreMigration := false
	t.Cleanup(func() {
		if restoreMigration {
			_ = manager.Up(context.Background(), plan)
		}
	})

	if err := manager.Up(ctx, plan); err != nil {
		t.Fatalf("up: %v", err)
	}

	version, hasVersion, dirty, err := manager.Version(ctx, plan)
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if version != 1 || !hasVersion || dirty {
		t.Fatalf(
			"version = %d, hasVersion = %t, dirty = %t",
			version,
			hasVersion,
			dirty,
		)
	}

	var sitesTable *string
	if err := connector.Pool().QueryRow(
		ctx,
		"SELECT to_regclass('core.sites')::text",
	).Scan(&sitesTable); err != nil {
		t.Fatal(err)
	}
	if sitesTable == nil || *sitesTable != "core.sites" {
		t.Fatalf("core.sites = %#v", sitesTable)
	}

	domain := "integration-test.example.com"
	seedPlan := seeds.Plan{
		Connection: string(connector.Code()),
		Target:     connector,
		Source: seeds.Source{
			ID:     "core",
			Schema: "core",
			FS: fstest.MapFS{
				"000001_integration_site.up.sql": {
					Data: []byte(`
INSERT INTO core.sites (profile_code, domain, locale, settings)
VALUES ('dev', 'integration-test.example.com', 'en-US', '{"enabled":true}'::jsonb)
ON CONFLICT DO NOTHING;
`),
				},
				"000001_integration_site.down.sql": {
					Data: []byte(`
DELETE FROM core.sites WHERE domain = 'integration-test.example.com';
`),
				},
			},
			Path: ".",
		},
	}
	seedManager := seeds.NewManager()
	if err := seedManager.Down(ctx, seedPlan, 1); err != nil {
		t.Fatalf("prepare seed state: %v", err)
	}
	if err := seedManager.Up(ctx, seedPlan); err != nil {
		t.Fatalf("seed up: %v", err)
	}

	sites, err := database.Sites().List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, item := range sites {
		if item.Domain != domain {
			continue
		}

		found = true
		rawSettings, err := json.Marshal(item.Settings)
		if err != nil {
			t.Fatal(err)
		}
		if string(rawSettings) != `{"enabled":true}` {
			t.Fatalf("settings = %s", rawSettings)
		}
	}
	if !found {
		t.Fatal("inserted site was not loaded")
	}

	if err := seedManager.Down(ctx, seedPlan, 1); err != nil {
		t.Fatalf("seed down: %v", err)
	}

	restoreMigration = true
	if err := manager.Down(ctx, plan, 1); err != nil {
		t.Fatalf("down: %v", err)
	}

	var schemaName *string
	var historyTable *string
	var seedHistoryTable *string
	if err := connector.Pool().QueryRow(ctx, `
SELECT
    to_regnamespace('core')::text,
    to_regclass('core.schema_migrations')::text,
    to_regclass('core.schema_seeds')::text;
`).Scan(&schemaName, &historyTable, &seedHistoryTable); err != nil {
		t.Fatal(err)
	}
	if schemaName == nil || *schemaName != "core" {
		t.Fatalf("core schema was removed: %#v", schemaName)
	}
	if historyTable == nil || *historyTable != "core.schema_migrations" {
		t.Fatalf("migration history was removed: %#v", historyTable)
	}
	if seedHistoryTable == nil || *seedHistoryTable != "core.schema_seeds" {
		t.Fatalf("seed history was removed: %#v", seedHistoryTable)
	}

	if err := manager.Up(ctx, plan); err != nil {
		t.Fatalf("restore up: %v", err)
	}
	restoreMigration = false
}
