package postgres

import (
	"embed"
	"errors"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

type Database struct {
	sites *SiteRepository
}

func NewDatabase(
	connector *connectorpostgres.Connector,
) (*Database, error) {
	if connector == nil {
		return nil, errors.New("postgres connector is nil")
	}

	sites, err := NewSiteRepository(connector)
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
var _ migrations.Provider = (*Database)(nil)
