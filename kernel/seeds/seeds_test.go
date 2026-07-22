package seeds_test

import (
	"context"
	"testing"
	"testing/fstest"

	migratedb "github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/stub"
	"github.com/vernal96/go-cms/kernel/seeds"
)

type target struct {
	driver       *stub.Stub
	historyTable string
}

func (t *target) OpenMigrationDriver(
	_ context.Context,
	_ string,
	historyTable string,
) (migratedb.Driver, error) {
	t.historyTable = historyTable
	return t.driver, nil
}

func TestSeedManagerUsesIndependentHistory(t *testing.T) {
	driver, err := stub.WithInstance(nil, &stub.Config{})
	if err != nil {
		t.Fatal(err)
	}

	target := &target{driver: driver.(*stub.Stub)}
	plan := seeds.Plan{
		Connection: "main",
		Target:     target,
		Source: seeds.Source{
			ID:     "core",
			Schema: "core",
			FS: fstest.MapFS{
				"000001_defaults.up.sql": {
					Data: []byte("INSERT DEFAULTS"),
				},
				"000001_defaults.down.sql": {
					Data: []byte("DELETE DEFAULTS"),
				},
			},
			Path: ".",
		},
	}

	manager := seeds.NewManager()
	if err := manager.Up(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if target.historyTable != seeds.DefaultHistoryTable {
		t.Fatalf("history table = %q", target.historyTable)
	}

	version, hasVersion, dirty, err := manager.Version(context.Background(), plan)
	if err != nil {
		t.Fatal(err)
	}
	if version != 1 || !hasVersion || dirty {
		t.Fatalf(
			"version = %d, hasVersion = %t, dirty = %t",
			version,
			hasVersion,
			dirty,
		)
	}

	if err := manager.Down(context.Background(), plan, 1); err != nil {
		t.Fatal(err)
	}
}
