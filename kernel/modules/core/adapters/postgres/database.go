package postgres

import (
	"embed"
	"errors"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	sitepostgres "github.com/vernal96/go-cms/kernel/modules/core/site/adapters/postgres"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Database struct {
	sites site.Repository
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

	return &Database{sites: sites}, nil
}

func (d *Database) ModuleCode() kernel.ModuleCode {
	return core.ModuleCode
}

func (d *Database) Sites() site.Repository {
	return d.sites
}

func (d *Database) MigrationSources() []migrations.Source {
	return []migrations.Source{
		{
			ID:     string(core.ModuleCode),
			Schema: "core",
			FS:     migrationFiles,
			Path:   "migrations",
		},
	}
}

var _ core.Database = (*Database)(nil)
var _ kernel.ModuleDatabaseFactory = DatabaseFactory{}
var _ migrations.Provider = (*Database)(nil)
