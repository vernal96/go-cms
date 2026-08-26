package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/access"
	accesspostgres "github.com/vernal96/go-cms/kernel/modules/core/access/adapters/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	filepostgres "github.com/vernal96/go-cms/kernel/modules/core/file/adapters/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/group"
	grouppostgres "github.com/vernal96/go-cms/kernel/modules/core/group/adapters/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	mediapostgres "github.com/vernal96/go-cms/kernel/modules/core/media/adapters/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	resourcepostgres "github.com/vernal96/go-cms/kernel/modules/core/resource/adapters/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	sitepostgres "github.com/vernal96/go-cms/kernel/modules/core/site/adapters/postgres"
	"github.com/vernal96/go-cms/kernel/modules/core/user"
	userpostgres "github.com/vernal96/go-cms/kernel/modules/core/user/adapters/postgres"
	"github.com/vernal96/go-cms/kernel/seeds"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

//go:embed seeds/shared/*.sql seeds/dev/*.sql
var seedFiles embed.FS

type Database struct {
	connector *connectorpostgres.Connector
	sites     site.Repository
	resources resource.Repository
	files     file.Repository
	media     media.Repository
	users     user.Repository
	groups    group.Repository
	access    access.Repository
}

type DatabaseFactory struct{}

func (DatabaseFactory) ModuleCode() kernel.ModuleCode {
	return core.ModuleCode
}

func (DatabaseFactory) Build(
	connector kernel.DBConnector,
) (kernel.ModuleDatabase, error) {
	postgresConnector, ok := connector.(*connectorpostgres.Connector)
	if !ok {
		return nil, errors.New(
			"core postgres adapter requires *postgres.Connector",
		)
	}

	return NewDatabase(postgresConnector)
}

func NewDatabase(
	connector *connectorpostgres.Connector,
) (*Database, error) {
	if connector == nil {
		return nil, errors.New("postgres connector is nil")
	}

	sites, err := sitepostgres.NewRepository(connector)
	if err != nil {
		return nil, err
	}

	resources, err := resourcepostgres.NewRepository(connector)
	if err != nil {
		return nil, err
	}

	files, err := filepostgres.NewRepository(connector)
	if err != nil {
		return nil, err
	}

	mediaRepository, err := mediapostgres.NewRepository(connector)
	if err != nil {
		return nil, err
	}
	userRepository, err := userpostgres.NewRepository(connector)
	if err != nil {
		return nil, err
	}
	groupRepository, err := grouppostgres.NewRepository(connector)
	if err != nil {
		return nil, err
	}
	accessRepository, err := accesspostgres.NewRepository(connector)
	if err != nil {
		return nil, err
	}

	return &Database{
		connector: connector,
		sites:     sites,
		resources: resources,
		files:     files,
		media:     mediaRepository,
		users:     userRepository,
		groups:    groupRepository,
		access:    accessRepository,
	}, nil
}

func (d *Database) ModuleCode() kernel.ModuleCode {
	return core.ModuleCode
}

func (d *Database) Sites() site.Repository {
	return d.sites
}

func (d *Database) Resources() resource.Repository {
	return d.resources
}

func (d *Database) Files() file.Repository {
	return d.files
}

func (d *Database) Media() media.Repository {
	return d.media
}

func (d *Database) Users() user.Repository {
	return d.users
}

func (d *Database) Groups() group.Repository {
	return d.groups
}

func (d *Database) Access() access.Repository {
	return d.access
}

func (d *Database) MigrationSources() []migrations.Source {
	return []migrations.Source{
		{
			ID:       string(core.ModuleCode),
			Schema:   "core",
			FS:       migrationFiles,
			Path:     "migrations",
			BeforeUp: d.beforeMigrations,
		},
	}
}

func (d *Database) beforeMigrations(ctx context.Context) error {
	if d == nil || d.connector == nil || d.connector.Pool() == nil {
		return errors.New("core migration preflight database is unavailable")
	}
	var hasSettings bool
	if err := d.connector.Pool().QueryRow(ctx, `
SELECT EXISTS (
 SELECT 1 FROM information_schema.columns
 WHERE table_schema='core' AND table_name='resources' AND column_name='settings'
);`).Scan(&hasSettings); err != nil {
		return fmt.Errorf("inspect legacy resource fields: %w", err)
	}
	if !hasSettings {
		return nil
	}
	var hasLegacyResources bool
	if err := d.connector.Pool().QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM core.resources);`).Scan(&hasLegacyResources); err != nil {
		return fmt.Errorf("inspect legacy resource settings: %w", err)
	}
	if !hasLegacyResources {
		return nil
	}
	var stagingReady bool
	if err := d.connector.Pool().QueryRow(ctx, `
SELECT to_regclass('core.resource_field_migration_manifest') IS NOT NULL
   AND to_regclass('core.resource_field_migration_rows') IS NOT NULL;`).Scan(&stagingReady); err != nil {
		return fmt.Errorf("inspect resource field migration staging: %w", err)
	}
	if !stagingReady {
		return errors.New("legacy resource settings require `resource-fields prepare` before migration 000015")
	}
	var incomplete bool
	if err := d.connector.Pool().QueryRow(ctx, `
SELECT EXISTS (
 SELECT 1 FROM core.resources resource
 LEFT JOIN core.resource_field_migration_manifest manifest ON manifest.resource_id=resource.id AND manifest.site_id=resource.site_id
	 WHERE manifest.resource_id IS NULL OR manifest.source_digest <> md5(resource.settings::text)
);`).Scan(&incomplete); err != nil {
		return fmt.Errorf("validate resource field migration staging: %w", err)
	}
	if incomplete {
		return errors.New("resource field migration staging is incomplete or stale; rerun `resource-fields prepare`")
	}
	return nil
}

func (d *Database) SeedSources() []seeds.Source {
	return []seeds.Source{
		{
			ID:     "identity_shared",
			Tags:   []seeds.Tag{"dev", "prod"},
			Schema: "core",
			FS:     seedFiles,
			Path:   "seeds/shared",
		},
		{
			ID:     "sites_dev",
			Tags:   []seeds.Tag{"dev"},
			Schema: "core",
			FS:     seedFiles,
			Path:   "seeds/dev",
		},
	}
}

var _ core.Database = (*Database)(nil)
var _ kernel.ModuleDatabaseFactory = DatabaseFactory{}
var _ migrations.Provider = (*Database)(nil)
var _ seeds.Provider = (*Database)(nil)
