package app_test

import (
	"context"
	"errors"
	"io/fs"
	"sync"
	"sync/atomic"
	"testing"
	"testing/fstest"

	migratedb "github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/stub"
	"github.com/vernal96/go-cms/app"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

const featureModuleCode kernel.ModuleCode = "feature"

type fakeConnector struct {
	code   kernel.ConnectionCode
	driver *stub.Stub
	pings  atomic.Int32
	closes atomic.Int32
}

func newFakeConnector(
	t *testing.T,
	code kernel.ConnectionCode,
) *fakeConnector {
	t.Helper()

	driver, err := stub.WithInstance(nil, &stub.Config{})
	if err != nil {
		t.Fatal(err)
	}

	return &fakeConnector{
		code:   code,
		driver: driver.(*stub.Stub),
	}
}

func (c *fakeConnector) Code() kernel.ConnectionCode {
	return c.code
}

func (c *fakeConnector) Ping(context.Context) error {
	c.pings.Add(1)
	return nil
}

func (c *fakeConnector) Close() error {
	c.closes.Add(1)
	return nil
}

func (c *fakeConnector) OpenMigrationDriver(
	context.Context,
	string,
	string,
) (migratedb.Driver, error) {
	return c.driver, nil
}

type fakeSiteRepository struct {
	mu         sync.Mutex
	sites      []site.Site
	err        error
	calls      int
	beforeList func()
}

func (r *fakeSiteRepository) List(
	context.Context,
) ([]site.Site, error) {
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

func (r *fakeSiteRepository) set(
	sites []site.Site,
	err error,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sites = append([]site.Site(nil), sites...)
	r.err = err
}

func (r *fakeSiteRepository) callCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.calls
}

type fakeCoreDatabase struct {
	repository site.Repository
	source     fs.FS
}

func (d *fakeCoreDatabase) ModuleCode() kernel.ModuleCode {
	return core.ModuleCode
}

func (d *fakeCoreDatabase) Sites() site.Repository {
	return d.repository
}

func (d *fakeCoreDatabase) MigrationSources() []migrations.Source {
	return []migrations.Source{
		{
			ID:     string(core.ModuleCode),
			Schema: "core",
			FS:     d.source,
			Path:   ".",
		},
	}
}

type fakeFeatureDatabase struct {
	name string
}

func (*fakeFeatureDatabase) ModuleCode() kernel.ModuleCode {
	return featureModuleCode
}

type featureConfig struct {
	Connection kernel.ConnectionCode
}

type featureModule struct {
	builds   *atomic.Int32
	selected **fakeFeatureDatabase
}

func (*featureModule) Code() kernel.ModuleCode {
	return featureModuleCode
}

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

type featureRuntime struct{}

func (featureRuntime) ModuleCode() kernel.ModuleCode {
	return featureModuleCode
}

func migrationFS() fs.FS {
	return fstest.MapFS{
		"000001_core.up.sql": &fstest.MapFile{
			Data: []byte("CREATE CORE"),
		},
		"000001_core.down.sql": &fstest.MapFile{
			Data: []byte("DROP CORE"),
		},
	}
}

func TestAppCompilesAndReusesRuntimes(t *testing.T) {
	ctx := context.Background()
	mainConnector := newFakeConnector(t, "main")
	logsConnector := newFakeConnector(t, "logs")

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
		if mainConnector.driver.CurrentVersion != 1 {
			t.Fatalf(
				"site repository called before migrations: version %d",
				mainConnector.driver.CurrentVersion,
			)
		}
	}

	mainFeature := &fakeFeatureDatabase{name: "main"}
	logsFeature := &fakeFeatureDatabase{name: "logs"}
	coreDatabase := &fakeCoreDatabase{
		repository: repository,
		source:     migrationFS(),
	}

	var builds atomic.Int32
	var selected *fakeFeatureDatabase
	module := &featureModule{
		builds:   &builds,
		selected: &selected,
	}

	profile := kernel.Profile{
		Code: "dev",
		Modules: []kernel.ProfileModule{
			{Module: core.Module{}},
			{
				Module: module,
				Config: featureConfig{Connection: "logs"},
			},
		},
	}

	runtimeApp, err := app.New(ctx, app.AppConfig{
		MainDatabase: app.DatabaseBinding{
			Connector: mainConnector,
			Adapters: []kernel.ModuleDatabase{
				coreDatabase,
				mainFeature,
			},
		},
		AdditionalDatabases: []app.DatabaseBinding{
			{
				Connector: logsConnector,
				Adapters:  []kernel.ModuleDatabase{logsFeature},
			},
		},
	}, []kernel.Profile{profile})
	if err != nil {
		t.Fatal(err)
	}

	if builds.Load() != 1 {
		t.Fatalf("feature Build calls = %d", builds.Load())
	}
	if selected != logsFeature {
		t.Fatalf("selected database = %#v, want logs database", selected)
	}
	if mainConnector.pings.Load() != 1 || logsConnector.pings.Load() != 1 {
		t.Fatalf(
			"ping counts = main:%d logs:%d",
			mainConnector.pings.Load(),
			logsConnector.pings.Load(),
		)
	}
	if repository.callCount() != 1 {
		t.Fatalf("repository calls after startup = %d", repository.callCount())
	}

	first, exists := runtimeApp.RuntimeByDomain("EXAMPLE.com.:443")
	if !exists {
		t.Fatal("runtime not found by Host with port")
	}
	second, exists := runtimeApp.RuntimeByDomain("example.com")
	if !exists || first != second {
		t.Fatal("same domain did not return the same runtime instance")
	}
	other, exists := runtimeApp.RuntimeByDomain("second.example.com")
	if !exists {
		t.Fatal("second runtime not found")
	}
	if first == other {
		t.Fatal("different sites share a site runtime")
	}
	if first.Profile() != other.Profile() {
		t.Fatal("sites of one profile do not share profile runtime")
	}
	if repository.callCount() != 1 {
		t.Fatalf("lookup called repository: calls = %d", repository.callCount())
	}

	item := first.Site()
	item.Settings["theme"] = "changed"
	if first.Site().Settings["theme"] != "light" {
		t.Fatal("site settings escaped runtime immutability")
	}

	configured := runtimeApp.Config()
	configured.MainDatabase.Adapters = nil
	if len(runtimeApp.Config().MainDatabase.Adapters) != 2 {
		t.Fatal("App.Config returned shared adapter slice")
	}

	repository.set([]site.Site{
		{
			ID:          3,
			ProfileCode: "dev",
			Domain:      "new.example.com",
			Locale:      "en-US",
		},
	}, nil)
	if err := runtimeApp.ReloadSites(ctx); err != nil {
		t.Fatalf("reload: %v", err)
	}

	current, exists := runtimeApp.RuntimeByDomain("new.example.com")
	if !exists {
		t.Fatal("new runtime not found after reload")
	}
	if _, exists := runtimeApp.RuntimeByDomain("example.com"); exists {
		t.Fatal("old domain remains after successful reload")
	}

	failedReloads := []struct {
		name  string
		sites []site.Site
		err   error
	}{
		{name: "repository error", err: errors.New("database unavailable")},
		{
			name: "unknown profile",
			sites: []site.Site{{
				ID: 4, ProfileCode: "unknown", Domain: "bad.example.com", Locale: "en-US",
			}},
		},
		{
			name: "duplicate normalized domain",
			sites: []site.Site{
				{ID: 5, ProfileCode: "dev", Domain: "duplicate.example.com", Locale: "en-US"},
				{ID: 6, ProfileCode: "dev", Domain: "DUPLICATE.example.com.", Locale: "en-US"},
			},
		},
		{
			name: "empty domain",
			sites: []site.Site{{
				ID: 7, ProfileCode: "dev", Domain: "", Locale: "en-US",
			}},
		},
	}

	for _, test := range failedReloads {
		t.Run(test.name, func(t *testing.T) {
			repository.set(test.sites, test.err)
			if err := runtimeApp.ReloadSites(ctx); err == nil {
				t.Fatal("expected reload error")
			}

			preserved, exists := runtimeApp.RuntimeByDomain("new.example.com")
			if !exists || preserved != current {
				t.Fatal("failed reload replaced the previous snapshot")
			}
		})
	}

	if err := runtimeApp.Close(); err != nil {
		t.Fatal(err)
	}
	if err := runtimeApp.Close(); err != nil {
		t.Fatal(err)
	}
	if mainConnector.closes.Load() != 1 || logsConnector.closes.Load() != 1 {
		t.Fatalf(
			"close counts = main:%d logs:%d",
			mainConnector.closes.Load(),
			logsConnector.closes.Load(),
		)
	}
	if _, exists := runtimeApp.RuntimeByDomain("new.example.com"); exists {
		t.Fatal("closed app still returns runtimes")
	}
}

func TestAppDoesNotRequirePostgresAdapter(t *testing.T) {
	connector := newFakeConnector(t, "mysql-main")
	repository := &fakeSiteRepository{}
	database := &fakeCoreDatabase{
		repository: repository,
		source:     migrationFS(),
	}

	runtimeApp, err := app.New(context.Background(), app.AppConfig{
		MainDatabase: app.DatabaseBinding{
			Connector: connector,
			Adapters:  []kernel.ModuleDatabase{database},
		},
	}, []kernel.Profile{
		{
			Code: "dev",
			Modules: []kernel.ProfileModule{
				{Module: core.Module{}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := runtimeApp.Close(); err != nil {
		t.Fatal(err)
	}
}
