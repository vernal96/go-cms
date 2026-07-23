package postgres

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"strconv"
	"testing"
	"time"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/seeds"
)

func TestDevSiteSeedSource(t *testing.T) {
	sources := (&Database{}).SeedSources()
	if len(sources) != 1 {
		t.Fatalf("seed sources = %#v", sources)
	}

	source := sources[0]
	if source.ID != "sites_dev" ||
		len(source.Tags) != 1 ||
		source.Tags[0] != "dev" ||
		source.Schema != "core" {
		t.Fatalf("dev site source = %#v", source)
	}
	if err := seeds.ValidateSource(source); err != nil {
		t.Fatal(err)
	}

	entries, err := fs.ReadDir(source.FS, source.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("dev seed files = %#v", entries)
	}
}

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

	seedPlan := seeds.Plan{
		Connection: string(connector.Code()),
		Module:     core.ModuleCode,
		Target:     connector,
		Source:     database.SeedSources()[0],
	}
	seedManager := seeds.NewManager()
	if err := seedManager.Force(ctx, seedPlan, -1); err != nil {
		t.Fatalf("prepare seed state: %v", err)
	}
	if _, err := connector.Pool().Exec(ctx, `
DELETE
FROM core.sites
WHERE profile_code = 'dev'
  AND domain IN ('localhost', 'example.com');
`); err != nil {
		t.Fatalf("clean seeded sites: %v", err)
	}
	if err := seedManager.Up(ctx, seedPlan); err != nil {
		t.Fatalf("seed up: %v", err)
	}

	loadedSites, err := database.Sites().List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	found := make(map[string]bool, 2)
	for _, item := range loadedSites {
		if item.Domain != "localhost" && item.Domain != "example.com" {
			continue
		}

		found[item.Domain] = true
		if item.ProfileCode != "dev" || item.Locale != "ru-RU" {
			t.Fatalf("seeded site = %#v", item)
		}
		rawSettings, err := json.Marshal(item.Settings)
		if err != nil {
			t.Fatal(err)
		}
		if string(rawSettings) != `{}` {
			t.Fatalf("settings = %s", rawSettings)
		}
	}
	if !found["localhost"] || !found["example.com"] {
		t.Fatalf("seeded domains = %#v", found)
	}

	if err := seedManager.Down(ctx, seedPlan, 1); err != nil {
		t.Fatalf("seed down: %v", err)
	}
	loadedSites, err = database.Sites().List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range loadedSites {
		if item.ProfileCode == "dev" &&
			(item.Domain == "localhost" || item.Domain == "example.com") {
			t.Fatalf("seed down kept site %#v", item)
		}
	}

	restoreMigration = true
	if err := manager.Down(ctx, plan, 1); err != nil {
		t.Fatalf("down: %v", err)
	}

	var schemaName *string
	var historyTable *string
	var devSeedHistoryTable *string
	if err := connector.Pool().QueryRow(ctx, `
SELECT
    to_regnamespace('core')::text,
    to_regclass('core.schema_migrations')::text,
    to_regclass('core.schema_seeds_sites_dev')::text;
`).Scan(
		&schemaName,
		&historyTable,
		&devSeedHistoryTable,
	); err != nil {
		t.Fatal(err)
	}
	if schemaName == nil || *schemaName != "core" {
		t.Fatalf("core schema was removed: %#v", schemaName)
	}
	if historyTable == nil || *historyTable != "core.schema_migrations" {
		t.Fatalf("migration history was removed: %#v", historyTable)
	}
	if devSeedHistoryTable == nil ||
		*devSeedHistoryTable != "core.schema_seeds_sites_dev" {
		t.Fatalf(
			"seed history was removed: %#v",
			devSeedHistoryTable,
		)
	}

	if err := manager.Up(ctx, plan); err != nil {
		t.Fatalf("restore up: %v", err)
	}
	restoreMigration = false
}
