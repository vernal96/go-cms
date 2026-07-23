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
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
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
	mu          sync.Mutex
	sites       []site.Site
	err         error
	updateErr   error
	calls       int
	updateCalls int
	beforeList  func()
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

func (r *fakeSiteRepository) UpdateSettings(
	_ context.Context,
	id site.ID,
	settings map[string]any,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.updateCalls++
	if r.updateErr != nil {
		return r.updateErr
	}
	if r.err != nil {
		return r.err
	}

	for index := range r.sites {
		if r.sites[index].ID != id {
			continue
		}

		r.sites[index].Settings = make(map[string]any, len(settings))
		for key, value := range settings {
			r.sites[index].Settings[key] = value
		}
		return nil
	}

	return site.ErrNotFound
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

func (r *fakeSiteRepository) setUpdateError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.updateErr = err
}

func (r *fakeSiteRepository) updateCallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.updateCalls
}

type fakeCoreDatabase struct {
	repository         site.Repository
	resourceRepository resource.Repository
	seedSources        []seeds.Source
}

func (*fakeCoreDatabase) ModuleCode() kernel.ModuleCode { return core.ModuleCode }
func (d *fakeCoreDatabase) Sites() site.Repository      { return d.repository }
func (d *fakeCoreDatabase) Resources() resource.Repository {
	if d.resourceRepository != nil {
		return d.resourceRepository
	}
	return fakeResourceRepository{}
}

type fakeResourceRepository struct{}

func (fakeResourceRepository) Create(
	context.Context,
	resource.Resource,
) (resource.Resource, error) {
	return resource.Resource{}, resource.ErrNotFound
}

func (fakeResourceRepository) ByID(
	context.Context,
	resource.ID,
) (resource.Resource, error) {
	return resource.Resource{}, resource.ErrNotFound
}

func (fakeResourceRepository) ByPath(
	context.Context,
	site.ID,
	string,
) (resource.Resource, error) {
	return resource.Resource{}, resource.ErrNotFound
}

func (fakeResourceRepository) ListBySite(
	context.Context,
	site.ID,
) ([]resource.Resource, error) {
	return nil, nil
}

func (fakeResourceRepository) Update(
	context.Context,
	resource.Resource,
) (resource.Resource, error) {
	return resource.Resource{}, resource.ErrNotFound
}

func (fakeResourceRepository) Delete(
	context.Context,
	resource.ID,
) error {
	return resource.ErrNotFound
}

type appResourceRepository struct {
	mu     sync.Mutex
	nextID resource.ID
	items  map[resource.ID]resource.Resource
}

func newAppResourceRepository() *appResourceRepository {
	return &appResourceRepository{
		nextID: 1,
		items:  make(map[resource.ID]resource.Resource),
	}
}

func (r *appResourceRepository) Create(
	_ context.Context,
	item resource.Resource,
) (resource.Resource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item = resource.Clone(item)
	item.ID = r.nextID
	r.nextID++
	r.items[item.ID] = item
	return resource.Clone(item), nil
}

func (r *appResourceRepository) ByID(
	_ context.Context,
	id resource.ID,
) (resource.Resource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, exists := r.items[id]
	if !exists {
		return resource.Resource{}, resource.ErrNotFound
	}
	return resource.Clone(item), nil
}

func (r *appResourceRepository) ByPath(
	_ context.Context,
	siteID site.ID,
	path string,
) (resource.Resource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range r.items {
		if item.SiteID == siteID &&
			item.Path != nil &&
			*item.Path == path {
			return resource.Clone(item), nil
		}
	}
	return resource.Resource{}, resource.ErrNotFound
}

func (r *appResourceRepository) ListBySite(
	_ context.Context,
	siteID site.ID,
) ([]resource.Resource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]resource.Resource, 0)
	for _, item := range r.items {
		if item.SiteID == siteID {
			result = append(result, resource.Clone(item))
		}
	}
	return result, nil
}

func (r *appResourceRepository) Update(
	_ context.Context,
	item resource.Resource,
) (resource.Resource, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.items[item.ID]; !exists {
		return resource.Resource{}, resource.ErrNotFound
	}
	r.items[item.ID] = resource.Clone(item)
	return resource.Clone(item), nil
}

func (r *appResourceRepository) Delete(
	_ context.Context,
	id resource.ID,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.items[id]; !exists {
		return resource.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

func (*fakeCoreDatabase) MigrationSources() []migrations.Source {
	return []migrations.Source{versionedSource("migration")}
}

func (d *fakeCoreDatabase) SeedSources() []seeds.Source {
	if d.seedSources != nil {
		return d.seedSources
	}

	return []seeds.Source{seedSource("defaults", "dev", "seed")}
}

type fakeFeatureDatabase struct {
	name        string
	seedSources []seeds.Source
}

func (*fakeFeatureDatabase) ModuleCode() kernel.ModuleCode {
	return featureModuleCode
}

func (d *fakeFeatureDatabase) SeedSources() []seeds.Source {
	return d.seedSources
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

func seedSource(
	id string,
	tag seeds.Tag,
	contents string,
) seeds.Source {
	source := versionedSource(contents)

	return seeds.Source{
		ID:     id,
		Tags:   []seeds.Tag{tag},
		Schema: source.Schema,
		FS:     source.FS,
		Path:   source.Path,
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
				Settings: map[string]any{
					"theme": "light",
					"roles": []any{"admin"},
				},
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
				Params: []field.Definition{
					{
						Key:   "theme",
						Type:  field.TypeString,
						Label: "Theme",
					},
					{
						Key:   "roles",
						Type:  field.TypeSelect,
						Label: "Roles",
						Options: field.SelectOptions{
							Multiple: true,
							Choices: []field.Choice{
								{Value: "admin", Label: "Admin"},
							},
						},
					},
				},
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
	if _, err := application.UpdateSiteSettings(
		ctx,
		1,
		map[string]any{"theme": "dark"},
	); !errors.Is(err, appkernel.ErrNotBooted) {
		t.Fatalf("UpdateSiteSettings before Boot = %v", err)
	}
	if _, exists := mainConnector.version(migrations.DefaultHistoryTable); exists {
		t.Fatal("New applied migrations")
	}
	if _, exists := mainConnector.version(
		seeds.HistoryTable("defaults"),
	); exists {
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
	firstCopy := first.Site()
	firstCopy.Settings["roles"].([]string)[0] = "mutated"
	if first.Site().Settings["roles"].([]string)[0] != "admin" {
		t.Fatal("site runtime exposed mutable settings slice")
	}

	_, err = application.UpdateSiteSettings(
		ctx,
		1,
		map[string]any{"unknown": "value"},
	)
	var validationErrors field.ValidationErrors
	if !errors.As(err, &validationErrors) {
		t.Fatalf("invalid settings error = %T %v", err, err)
	}
	unchanged, exists := application.RuntimeByDomain("example.com")
	if !exists || unchanged != first {
		t.Fatal("validation failure changed site runtime")
	}
	if repository.updateCallCount() != 0 {
		t.Fatal("validation failure called site repository")
	}

	updated, err := application.UpdateSiteSettings(
		ctx,
		1,
		map[string]any{"theme": "dark"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Site().Settings["theme"] != "dark" {
		t.Fatalf("updated settings = %#v", updated.Site().Settings)
	}
	currentByDomain, exists := application.RuntimeByDomain("example.com")
	if !exists || currentByDomain != updated || currentByDomain == first {
		t.Fatal("successful settings update did not replace runtime")
	}
	if repository.updateCallCount() != 1 {
		t.Fatalf("repository update calls = %d", repository.updateCallCount())
	}

	repository.setUpdateError(errors.New("update unavailable"))
	_, err = application.UpdateSiteSettings(
		ctx,
		1,
		map[string]any{"theme": "broken"},
	)
	if err == nil || !strings.Contains(err.Error(), "update unavailable") {
		t.Fatalf("repository update error = %v", err)
	}
	preservedAfterUpdateError, exists := application.RuntimeByDomain(
		"example.com",
	)
	if !exists || preservedAfterUpdateError != updated {
		t.Fatal("repository failure changed site runtime")
	}
	repository.setUpdateError(nil)

	if _, err := application.UpdateSiteSettings(
		ctx,
		999,
		map[string]any{"theme": "missing"},
	); !errors.Is(err, site.ErrNotFound) {
		t.Fatalf("missing site update error = %v", err)
	}

	consoleOutput.Reset()
	if err := application.Console().Run(
		ctx,
		[]string{"seeds", "up", "-tags=dev"},
		console.IO{Out: &consoleOutput},
	); err != nil {
		t.Fatal(err)
	}
	seedVersion, exists := mainConnector.version(
		seeds.HistoryTable("defaults"),
	)
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

	repository.set([]site.Site{
		{
			ID:          4,
			ProfileCode: "dev",
			Domain:      "invalid.example.com",
			Locale:      "en-US",
			Settings:    map[string]any{"unknown": true},
		},
	}, nil)
	if err := application.ReloadSites(ctx); err == nil {
		t.Fatal("expected invalid stored settings error")
	}
	preserved, exists := application.RuntimeByDomain("new.example.com")
	if !exists || preserved != current {
		t.Fatal("invalid settings reload replaced the previous snapshot")
	}

	repository.set(nil, errors.New("database unavailable"))
	if err := application.ReloadSites(ctx); err == nil {
		t.Fatal("expected reload error")
	}
	preserved, exists = application.RuntimeByDomain("new.example.com")
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

func TestAppResourceFacades(t *testing.T) {
	ctx := context.Background()
	connector := newFakeConnector("main")
	resourceRepository := newAppResourceRepository()
	coreDatabase := &fakeCoreDatabase{
		repository: &fakeSiteRepository{
			sites: []site.Site{{
				ID:          1,
				ProfileCode: "dev",
				Domain:      "example.com",
				Locale:      "en-US",
			}},
		},
		resourceRepository: resourceRepository,
	}
	templateCode := template.Code("article")

	application, err := appkernel.New(ctx, appkernel.Definition{
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: &fakeConnectorFactory{connector: connector},
			Adapters: []kernel.ModuleDatabaseFactory{
				&fakeDatabaseFactory{
					code:     core.ModuleCode,
					database: coreDatabase,
				},
			},
		},
		Profiles: []kernel.Profile{{
			Code: "dev",
			Modules: []kernel.ProfileModule{{
				Module: core.Module{},
			}},
			Templates: []template.Definition{{
				Code:  templateCode,
				Label: "Article",
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = application.Close() }()

	if _, err := application.CreateResource(
		ctx,
		resource.CreateInput{},
	); !errors.Is(err, appkernel.ErrNotBooted) {
		t.Fatalf("create before boot error = %v", err)
	}
	if err := application.Boot(ctx); err != nil {
		t.Fatal(err)
	}

	created, err := application.CreateResource(
		ctx,
		resource.CreateInput{
			SiteID:   1,
			Template: &templateCode,
			Title:    "Home",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if created.Type != resourcetype.Page ||
		created.Path == nil ||
		*created.Path != "/" {
		t.Fatalf("created resource = %#v", created)
	}

	byID, err := application.Resource(ctx, created.ID)
	if err != nil || byID.ID != created.ID {
		t.Fatalf("resource by id = %#v, %v", byID, err)
	}
	byPath, err := application.ResourceByPath(ctx, 1, "/")
	if err != nil || byPath.ID != created.ID {
		t.Fatalf("resource by path = %#v, %v", byPath, err)
	}
	tree, err := application.ResourceTree(ctx, 1)
	if err != nil || len(tree) != 1 ||
		tree[0].Resource.ID != created.ID {
		t.Fatalf("resource tree = %#v, %v", tree, err)
	}

	updated, err := application.UpdateResource(
		ctx,
		resource.UpdateInput{
			ID:           created.ID,
			Type:         resourcetype.Page,
			Template:     &templateCode,
			Title:        "Updated home",
			IsPublic:     true,
			IsSearchable: true,
			InMenu:       true,
			InSitemap:    true,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "Updated home" {
		t.Fatalf("updated resource = %#v", updated)
	}

	if err := application.DeleteResource(ctx, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := application.Resource(
		ctx,
		created.ID,
	); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("deleted resource error = %v", err)
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

func TestAppCollectsSeedSourcesAcrossConnectionsAndClonesTags(t *testing.T) {
	mainConnector := newFakeConnector("main")
	logsConnector := newFakeConnector("logs")
	coreDatabase := &fakeCoreDatabase{
		repository: &fakeSiteRepository{},
		seedSources: []seeds.Source{
			seedSource("sites_dev", "dev", "core dev"),
		},
	}
	mainFeature := &fakeFeatureDatabase{
		name: "main",
		seedSources: []seeds.Source{
			{
				ID:     "feature_shared",
				Tags:   []seeds.Tag{"dev", "prod"},
				Schema: "feature",
				FS:     versionedSource("feature shared").FS,
				Path:   ".",
			},
		},
	}
	logsFeature := &fakeFeatureDatabase{
		name: "logs",
		seedSources: []seeds.Source{
			{
				ID:     "audit_prod",
				Tags:   []seeds.Tag{"prod"},
				Schema: "feature",
				FS:     versionedSource("audit prod").FS,
				Path:   ".",
			},
		},
	}

	application, err := appkernel.New(
		context.Background(),
		appkernel.Definition{
			MainDatabase: appkernel.DatabaseDefinition{
				Connector: &fakeConnectorFactory{connector: mainConnector},
				Adapters: []kernel.ModuleDatabaseFactory{
					&fakeDatabaseFactory{
						code:     core.ModuleCode,
						database: coreDatabase,
					},
					&fakeDatabaseFactory{
						code:     featureModuleCode,
						database: mainFeature,
					},
				},
			},
			AdditionalDatabases: []appkernel.DatabaseDefinition{
				{
					Connector: &fakeConnectorFactory{
						connector: logsConnector,
					},
					Adapters: []kernel.ModuleDatabaseFactory{
						&fakeDatabaseFactory{
							code:     featureModuleCode,
							database: logsFeature,
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
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = application.Close() }()

	plans := application.SeedPlans()
	if len(plans) != 3 {
		t.Fatalf("seed plans = %#v", plans)
	}
	got := []string{
		plans[0].Connection + "/" +
			string(plans[0].Module) + "/" +
			plans[0].Source.ID,
		plans[1].Connection + "/" +
			string(plans[1].Module) + "/" +
			plans[1].Source.ID,
		plans[2].Connection + "/" +
			string(plans[2].Module) + "/" +
			plans[2].Source.ID,
	}
	want := []string{
		"main/core/sites_dev",
		"main/feature/feature_shared",
		"logs/feature/audit_prod",
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("plan order = %#v", got)
		}
	}

	plans[1].Source.Tags[0] = "changed"
	fresh := application.SeedPlans()
	if fresh[1].Source.Tags[0] != "dev" {
		t.Fatalf("seed plan tags were not cloned: %#v", fresh[1].Source.Tags)
	}
}

func TestAppRejectsSeedHistoryCollision(t *testing.T) {
	connector := newFakeConnector("main")
	coreSource := seedSource("shared", "dev", "core")
	coreSource.Schema = "shared"
	featureSource := seedSource("shared", "prod", "feature")
	featureSource.Schema = "shared"

	_, err := appkernel.New(
		context.Background(),
		appkernel.Definition{
			MainDatabase: appkernel.DatabaseDefinition{
				Connector: &fakeConnectorFactory{connector: connector},
				Adapters: []kernel.ModuleDatabaseFactory{
					&fakeDatabaseFactory{
						code: core.ModuleCode,
						database: &fakeCoreDatabase{
							repository:  &fakeSiteRepository{},
							seedSources: []seeds.Source{coreSource},
						},
					},
					&fakeDatabaseFactory{
						code: featureModuleCode,
						database: &fakeFeatureDatabase{
							seedSources: []seeds.Source{featureSource},
						},
					},
				},
			},
		},
	)
	if err == nil || !strings.Contains(err.Error(), "share history") {
		t.Fatalf("history collision error = %v", err)
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
