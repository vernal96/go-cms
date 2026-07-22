package app_test

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"

	migratedb "github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/stub"
	"github.com/vernal96/go-cms/kernel"
	appkernel "github.com/vernal96/go-cms/kernel/app"
	"github.com/vernal96/go-cms/kernel/console"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/seeds"
)

const featureModuleCode kernel.ModuleCode = "feature"

type fakeConnector struct {
	code kernel.ConnectionCode

	mu      sync.Mutex
	drivers map[string]*stub.Stub
	pings   atomic.Int32
	closes  atomic.Int32
}

func newFakeConnector(code kernel.ConnectionCode) *fakeConnector {
	return &fakeConnector{
		code:    code,
		drivers: make(map[string]*stub.Stub),
	}
}

func (c *fakeConnector) Code() kernel.ConnectionCode { return c.code }

func (c *fakeConnector) Ping(context.Context) error {
	c.pings.Add(1)
	return nil
}

func (c *fakeConnector) Close() error {
	c.closes.Add(1)
	return nil
}

func (c *fakeConnector) OpenMigrationDriver(
	_ context.Context,
	_ string,
	historyTable string,
) (migratedb.Driver, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if driver, exists := c.drivers[historyTable]; exists {
		return driver, nil
	}

	driver, err := stub.WithInstance(nil, &stub.Config{})
	if err != nil {
		return nil, err
	}

	c.drivers[historyTable] = driver.(*stub.Stub)
	return driver, nil
}

func (c *fakeConnector) version(historyTable string) (int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	driver, exists := c.drivers[historyTable]
	if !exists {
		return 0, false
	}
	return driver.CurrentVersion, true
}

type fakeConnectorFactory struct {
	connector *fakeConnector
	opens     atomic.Int32
	err       error
}

func (f *fakeConnectorFactory) Code() kernel.ConnectionCode {
	return f.connector.code
}

func (f *fakeConnectorFactory) Open(
	context.Context,
) (kernel.DBConnector, error) {
	f.opens.Add(1)
	return f.connector, f.err
}

type fakeSiteRepository struct {
	mu         sync.Mutex
	sites      []site.Site
	err        error
	calls      int
	beforeList func()
}

func (r *fakeSiteRepository) List(context.Context) ([]site.Site, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.calls++
	if r.beforeList != nil {
		r.beforeList()
	}
	if r.err != nil {
		return nil, r.err
	}

	return append([]site.Site(nil), r.sites...), nil
}

func (r *fakeSiteRepository) set(items []site.Site, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sites = append([]site.Site(nil), items...)
	r.err = err
}

func (r *fakeSiteRepository) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

type fakeCoreDatabase struct {
	repository site.Repository
}

func (*fakeCoreDatabase) ModuleCode() kernel.ModuleCode { return core.ModuleCode }
func (d *fakeCoreDatabase) Sites() site.Repository      { return d.repository }

func (*fakeCoreDatabase) MigrationSources() []migrations.Source {
	return []migrations.Source{versionedSource("migration")}
}

func (*fakeCoreDatabase) SeedSources() []seeds.Source {
	return []seeds.Source{versionedSource("seed")}
}

type fakeFeatureDatabase struct {
	name string
}

func (*fakeFeatureDatabase) ModuleCode() kernel.ModuleCode {
	return featureModuleCode
}

type fakeDatabaseFactory struct {
	code     kernel.ModuleCode
	database kernel.ModuleDatabase
	err      error
	builds   atomic.Int32
}

func (f *fakeDatabaseFactory) ModuleCode() kernel.ModuleCode { return f.code }

func (f *fakeDatabaseFactory) Build(
	kernel.DBConnector,
) (kernel.ModuleDatabase, error) {
	f.builds.Add(1)
	return f.database, f.err
}

type featureConfig struct {
	Connection kernel.ConnectionCode
}

type featureModule struct {
	builds   *atomic.Int32
	selected **fakeFeatureDatabase
}

func (*featureModule) Code() kernel.ModuleCode { return featureModuleCode }

func (m *featureModule) Build(
	_ context.Context,
	ctx kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	m.builds.Add(1)

	config, err := kernel.ModuleConfigFrom[featureConfig](ctx)
	if err != nil {
		return nil, err
	}
	database, err := kernel.ModuleDatabaseFrom[*fakeFeatureDatabase](
		ctx,
		config.Connection,
		featureModuleCode,
	)
	if err != nil {
		return nil, err
	}

	*m.selected = database
	return featureRuntime{}, nil
}

func (*featureModule) Commands() []console.Command {
	return []console.Command{testCommand{name: "feature-info"}}
}

type featureRuntime struct{}

func (featureRuntime) ModuleCode() kernel.ModuleCode { return featureModuleCode }

type testCommand struct{ name string }

func (c testCommand) Name() string      { return c.name }
func (testCommand) Description() string { return "test feature command" }
func (c testCommand) Run(
	_ context.Context,
	_ []string,
	streams console.IO,
) error {
	_, err := streams.Out.Write([]byte(c.name))
	return err
}

func versionedSource(contents string) migrations.Source {
	return migrations.Source{
		ID:     string(core.ModuleCode),
		Schema: "core",
		FS: fstest.MapFS{
			"000001_core.up.sql": {
				Data: []byte("UP " + contents),
			},
			"000001_core.down.sql": {
				Data: []byte("DOWN " + contents),
			},
		},
		Path: ".",
	}
}

func TestAppNewBootConsoleAndRuntimeLifecycle(t *testing.T) {
	ctx := context.Background()
	mainConnector := newFakeConnector("main")
	logsConnector := newFakeConnector("logs")

	repository := &fakeSiteRepository{
		sites: []site.Site{
			{
				ID:          1,
				ProfileCode: "dev",
				Domain:      "Example.COM.",
				Locale:      "ru-RU",
				Settings:    map[string]any{"theme": "light"},
			},
			{
				ID:          2,
				ProfileCode: "dev",
				Domain:      "second.example.com",
				Locale:      "ru-RU",
			},
		},
	}
	repository.beforeList = func() {
		version, exists := mainConnector.version(migrations.DefaultHistoryTable)
		if !exists || version != 1 {
			t.Fatalf("repository called before migration up: %d, %t", version, exists)
		}
	}

	mainFeature := &fakeFeatureDatabase{name: "main"}
	logsFeature := &fakeFeatureDatabase{name: "logs"}
	coreDatabase := &fakeCoreDatabase{repository: repository}

	var moduleBuilds atomic.Int32
	var selected *fakeFeatureDatabase
	module := &featureModule{builds: &moduleBuilds, selected: &selected}

	application, err := appkernel.New(ctx, appkernel.Definition{
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: &fakeConnectorFactory{connector: mainConnector},
			Adapters: []kernel.ModuleDatabaseFactory{
				&fakeDatabaseFactory{code: core.ModuleCode, database: coreDatabase},
				&fakeDatabaseFactory{code: featureModuleCode, database: mainFeature},
			},
		},
		AdditionalDatabases: []appkernel.DatabaseDefinition{
			{
				Connector: &fakeConnectorFactory{connector: logsConnector},
				Adapters: []kernel.ModuleDatabaseFactory{
					&fakeDatabaseFactory{code: featureModuleCode, database: logsFeature},
				},
			},
		},
		Profiles: []kernel.Profile{
			{
				Code: "dev",
				Modules: []kernel.ProfileModule{
					{Module: core.Module{}},
					{
						Module: module,
						Config: featureConfig{Connection: "logs"},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if mainConnector.pings.Load() != 1 || logsConnector.pings.Load() != 1 {
		t.Fatalf(
			"ping counts = main:%d logs:%d",
			mainConnector.pings.Load(),
			logsConnector.pings.Load(),
		)
	}
	if repository.callCount() != 0 {
		t.Fatalf("New called site repository %d times", repository.callCount())
	}
	if _, exists := application.RuntimeByDomain("example.com"); exists {
		t.Fatal("runtime exists before Boot")
	}
	if err := application.ReloadSites(ctx); !errors.Is(err, appkernel.ErrNotBooted) {
		t.Fatalf("ReloadSites before Boot = %v", err)
	}
	if _, exists := mainConnector.version(migrations.DefaultHistoryTable); exists {
		t.Fatal("New applied migrations")
	}
	if _, exists := mainConnector.version(seeds.DefaultHistoryTable); exists {
		t.Fatal("New applied seeds")
	}
	consoleDatabase, exists := application.Console().Application().ModuleDatabase(
		"logs",
		featureModuleCode,
	)
	if !exists || consoleDatabase != logsFeature {
		t.Fatalf("console resolved database = %#v", consoleDatabase)
	}

	var consoleOutput bytes.Buffer
	if err := application.Console().Run(
		ctx,
		[]string{"feature-info"},
		console.IO{Out: &consoleOutput},
	); err != nil {
		t.Fatal(err)
	}
	if consoleOutput.String() != "feature-info" {
		t.Fatalf("custom command output = %q", consoleOutput.String())
	}

	consoleOutput.Reset()
	if err := application.Console().Run(
		ctx,
		[]string{"migrations", "status"},
		console.IO{Out: &consoleOutput},
	); err != nil {
		t.Fatal(err)
	}
	mainConnector.mu.Lock()
	mainConnector.drivers[migrations.DefaultHistoryTable].CurrentVersion = 1
	mainConnector.drivers[migrations.DefaultHistoryTable].IsDirty = true
	mainConnector.mu.Unlock()

	consoleOutput.Reset()
	if err := application.Console().Run(
		ctx,
		[]string{
			"migrations",
			"force",
			"-connection=main",
			"-module=core",
			"-version=-1",
		},
		console.IO{Out: &consoleOutput},
	); err != nil {
		t.Fatalf("repair dirty migration before Boot: %v", err)
	}

	if err := application.Boot(ctx); err != nil {
		t.Fatal(err)
	}
	if err := application.Boot(ctx); err != nil {
		t.Fatal(err)
	}
	if moduleBuilds.Load() != 1 {
		t.Fatalf("module Build calls = %d", moduleBuilds.Load())
	}
	if selected != logsFeature {
		t.Fatalf("selected database = %#v", selected)
	}
	if repository.callCount() != 1 {
		t.Fatalf("repository calls after Boot = %d", repository.callCount())
	}
	if _, exists := mainConnector.version(seeds.DefaultHistoryTable); exists {
		t.Fatal("Boot applied seeds")
	}

	first, exists := application.RuntimeByDomain("EXAMPLE.com.:443")
	if !exists {
		t.Fatal("runtime not found by Host with port")
	}
	second, exists := application.RuntimeByDomain("example.com")
	if !exists || first != second {
		t.Fatal("same domain did not return the same runtime")
	}
	other, exists := application.RuntimeByDomain("second.example.com")
	if !exists || other == first {
		t.Fatal("different site did not get a distinct runtime")
	}
	if first.Profile() != other.Profile() {
		t.Fatal("sites of one profile do not share profile runtime")
	}

	consoleOutput.Reset()
	if err := application.Console().Run(
		ctx,
		[]string{"seeds", "up"},
		console.IO{Out: &consoleOutput},
	); err != nil {
		t.Fatal(err)
	}
	seedVersion, exists := mainConnector.version(seeds.DefaultHistoryTable)
	if !exists || seedVersion != 1 {
		t.Fatalf("seed version = %d, exists = %t", seedVersion, exists)
	}

	repository.set([]site.Site{
		{ID: 3, ProfileCode: "dev", Domain: "new.example.com", Locale: "en-US"},
	}, nil)
	if err := application.ReloadSites(ctx); err != nil {
		t.Fatal(err)
	}
	current, exists := application.RuntimeByDomain("new.example.com")
	if !exists {
		t.Fatal("reloaded runtime not found")
	}

	repository.set(nil, errors.New("database unavailable"))
	if err := application.ReloadSites(ctx); err == nil {
		t.Fatal("expected reload error")
	}
	preserved, exists := application.RuntimeByDomain("new.example.com")
	if !exists || preserved != current {
		t.Fatal("failed reload replaced the previous snapshot")
	}

	if err := application.Close(); err != nil {
		t.Fatal(err)
	}
	if err := application.Close(); err != nil {
		t.Fatal(err)
	}
	if mainConnector.closes.Load() != 1 || logsConnector.closes.Load() != 1 {
		t.Fatalf(
			"close counts = main:%d logs:%d",
			mainConnector.closes.Load(),
			logsConnector.closes.Load(),
		)
	}
}

func TestNewClosesPreviouslyOpenedConnectorOnFactoryError(t *testing.T) {
	mainConnector := newFakeConnector("main")
	brokenConnector := newFakeConnector("broken")

	_, err := appkernel.New(context.Background(), appkernel.Definition{
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: &fakeConnectorFactory{connector: mainConnector},
			Adapters: []kernel.ModuleDatabaseFactory{
				&fakeDatabaseFactory{
					code: core.ModuleCode,
					database: &fakeCoreDatabase{
						repository: &fakeSiteRepository{},
					},
				},
			},
		},
		AdditionalDatabases: []appkernel.DatabaseDefinition{
			{
				Connector: &fakeConnectorFactory{
					connector: brokenConnector,
					err:       errors.New("open failed"),
				},
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "open failed") {
		t.Fatalf("New error = %v", err)
	}
	if mainConnector.closes.Load() != 1 || brokenConnector.closes.Load() != 1 {
		t.Fatalf(
			"close counts = main:%d broken:%d",
			mainConnector.closes.Load(),
			brokenConnector.closes.Load(),
		)
	}
}

func TestBootFailureIsRememberedAndNotRetried(t *testing.T) {
	connector := newFakeConnector("main")
	repository := &fakeSiteRepository{err: errors.New("list failed")}

	application, err := appkernel.New(context.Background(), appkernel.Definition{
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: &fakeConnectorFactory{connector: connector},
			Adapters: []kernel.ModuleDatabaseFactory{
				&fakeDatabaseFactory{
					code: core.ModuleCode,
					database: &fakeCoreDatabase{
						repository: repository,
					},
				},
			},
		},
		Profiles: []kernel.Profile{
			{
				Code: "dev",
				Modules: []kernel.ProfileModule{
					{Module: core.Module{}},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = application.Close() }()

	firstErr := application.Boot(context.Background())
	if firstErr == nil {
		t.Fatal("expected Boot error")
	}

	repository.set(nil, nil)
	secondErr := application.Boot(context.Background())
	if secondErr == nil || secondErr.Error() != firstErr.Error() {
		t.Fatalf("second Boot error = %v, want %v", secondErr, firstErr)
	}
	if repository.callCount() != 1 {
		t.Fatalf("failed Boot was retried: repository calls = %d", repository.callCount())
	}
	if _, exists := application.ProfileRuntime("dev"); exists {
		t.Fatal("failed Boot published profile runtime")
	}
}

var _ fs.FS = fstest.MapFS{}
