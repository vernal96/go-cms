package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/console"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/seeds"
)

var (
	ErrClosed    = errors.New("app is closed")
	ErrNotBooted = errors.New("app is not booted")
)

type ConnectorFactory = kernel.ConnectorFactory
type ModuleDatabaseFactory = kernel.ModuleDatabaseFactory

type DatabaseDefinition struct {
	Connector ConnectorFactory
	Adapters  []ModuleDatabaseFactory
}

type Definition struct {
	MainDatabase        DatabaseDefinition
	AdditionalDatabases []DatabaseDefinition
	Profiles            []kernel.Profile
}

type bindingRuntime struct {
	connector kernel.DBConnector
	adapters  map[kernel.ModuleCode]kernel.ModuleDatabase
}

type App struct {
	definition Definition

	main          *bindingRuntime
	additional    map[kernel.ConnectionCode]*bindingRuntime
	connectors    []kernel.DBConnector
	coreDatabase  core.Database
	migrationPlan []migrations.Plan
	seedPlan      []seeds.Plan
	providers     []console.Provider
	console       *console.Console

	profileRuntimes map[kernel.ProfileCode]*kernel.ProfileRuntime
	sites           *site.Catalog

	bootOnce sync.Once
	bootErr  error
	booted   atomic.Bool
	closed   atomic.Bool

	lifecycleMu sync.RWMutex
	closeOnce   sync.Once
	closeErr    error
}

func New(
	ctx context.Context,
	definition Definition,
) (_ *App, resultErr error) {
	if ctx == nil {
		return nil, errors.New("app context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	definition = cloneDefinition(definition)
	if err := validateDefinition(definition); err != nil {
		return nil, err
	}

	application := &App{
		definition: definition,
		additional: make(map[kernel.ConnectionCode]*bindingRuntime),
	}

	defer func() {
		if resultErr == nil {
			return
		}

		resultErr = errors.Join(resultErr, application.Close())
	}()

	definitions := make(
		[]DatabaseDefinition,
		0,
		len(definition.AdditionalDatabases)+1,
	)
	definitions = append(definitions, definition.MainDatabase)
	definitions = append(definitions, definition.AdditionalDatabases...)

	for index, databaseDefinition := range definitions {
		binding, err := application.openBinding(ctx, databaseDefinition)
		if err != nil {
			return nil, err
		}

		if index == 0 {
			application.main = binding
		} else {
			application.additional[binding.connector.Code()] = binding
		}
	}

	coreAdapter, exists := application.main.adapters[core.ModuleCode]
	if !exists {
		return nil, errors.New(
			"main database binding does not contain core database",
		)
	}

	coreDatabase, ok := coreAdapter.(core.Database)
	if !ok {
		return nil, fmt.Errorf(
			"main database adapter %q does not implement core.Database",
			core.ModuleCode,
		)
	}
	if coreDatabase.Sites() == nil {
		return nil, errors.New("main core database has nil site repository")
	}
	application.coreDatabase = coreDatabase

	application.collectModuleCommandProviders()

	runner, err := console.New(application)
	if err != nil {
		return nil, err
	}
	application.console = runner

	return application, nil
}

func (a *App) Boot(ctx context.Context) error {
	if a == nil {
		return errors.New("app is nil")
	}

	a.bootOnce.Do(func() {
		a.bootErr = a.boot(ctx)
	})

	return a.bootErr
}

func (a *App) boot(ctx context.Context) error {
	if ctx == nil {
		return errors.New("app boot context is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()

	if a.closed.Load() {
		return ErrClosed
	}

	if err := migrations.NewManager().UpAll(ctx, a.migrationPlan); err != nil {
		return err
	}

	factory, err := kernel.NewProfileRuntimeFactory(a)
	if err != nil {
		return err
	}

	profileRuntimes := make(
		map[kernel.ProfileCode]*kernel.ProfileRuntime,
		len(a.definition.Profiles),
	)

	for _, profile := range a.definition.Profiles {
		runtime, err := factory.Make(ctx, profile)
		if err != nil {
			return fmt.Errorf(
				"compile profile runtime %q: %w",
				profile.Code,
				err,
			)
		}

		if _, exists := runtime.Registry().Module(core.ModuleCode); !exists {
			return fmt.Errorf(
				"profile %q does not contain required module %q",
				profile.Code,
				core.ModuleCode,
			)
		}

		profileRuntimes[profile.Code] = runtime
	}

	catalog, err := site.NewCatalog(
		a.coreDatabase.Sites(),
		profileResolver(profileRuntimes),
	)
	if err != nil {
		return err
	}

	if err := catalog.Reload(ctx); err != nil {
		return fmt.Errorf("compile site runtimes: %w", err)
	}

	a.profileRuntimes = profileRuntimes
	a.sites = catalog
	a.booted.Store(true)
	return nil
}

func (a *App) Definition() Definition {
	if a == nil {
		return Definition{}
	}

	return cloneDefinition(a.definition)
}

func (a *App) Console() *console.Console {
	if a == nil {
		return nil
	}

	return a.console
}

func (a *App) ProfileRuntime(
	code kernel.ProfileCode,
) (*kernel.ProfileRuntime, bool) {
	if a == nil || a.closed.Load() || !a.booted.Load() {
		return nil, false
	}

	runtime, exists := a.profileRuntimes[code]
	return runtime, exists
}

func (a *App) RuntimeByDomain(
	domain string,
) (*site.Runtime, bool) {
	if a == nil || a.closed.Load() || !a.booted.Load() {
		return nil, false
	}

	return a.sites.RuntimeByDomain(domain)
}

func (a *App) ReloadSites(ctx context.Context) error {
	if a == nil {
		return errors.New("app is nil")
	}
	if ctx == nil {
		return errors.New("site reload context is nil")
	}
	if !a.booted.Load() {
		return ErrNotBooted
	}

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()

	if a.closed.Load() {
		return ErrClosed
	}

	return a.sites.Reload(ctx)
}

func (a *App) MigrationPlans() []migrations.Plan {
	if a == nil {
		return nil
	}

	return append([]migrations.Plan(nil), a.migrationPlan...)
}

func (a *App) SeedPlans() []seeds.Plan {
	if a == nil {
		return nil
	}

	return append([]seeds.Plan(nil), a.seedPlan...)
}

func (a *App) CommandProviders() []console.Provider {
	if a == nil {
		return nil
	}

	return append([]console.Provider(nil), a.providers...)
}

func (a *App) MainModuleDatabase(
	moduleCode kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	if a == nil || a.main == nil || a.closed.Load() {
		return nil, false
	}

	database, exists := a.main.adapters[moduleCode]
	return database, exists
}

func (a *App) ModuleDatabase(
	connectionCode kernel.ConnectionCode,
	moduleCode kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	if a == nil || a.closed.Load() {
		return nil, false
	}

	if connectionCode == "" || connectionCode == a.main.connector.Code() {
		return a.MainModuleDatabase(moduleCode)
	}

	binding, exists := a.additional[connectionCode]
	if !exists {
		return nil, false
	}

	database, exists := binding.adapters[moduleCode]
	return database, exists
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}

	a.closeOnce.Do(func() {
		a.lifecycleMu.Lock()
		defer a.lifecycleMu.Unlock()

		a.closed.Store(true)

		var closeErrors []error
		for index := len(a.connectors) - 1; index >= 0; index-- {
			connector := a.connectors[index]
			if err := connector.Close(); err != nil {
				closeErrors = append(closeErrors, fmt.Errorf(
					"close database connector %q: %w",
					connector.Code(),
					err,
				))
			}
		}

		a.closeErr = errors.Join(closeErrors...)
	})

	return a.closeErr
}

func (a *App) openBinding(
	ctx context.Context,
	definition DatabaseDefinition,
) (*bindingRuntime, error) {
	connector, err := definition.Connector.Open(ctx)
	if connector != nil {
		a.connectors = append(a.connectors, connector)
	}
	if err != nil {
		return nil, fmt.Errorf(
			"open database connector %q: %w",
			definition.Connector.Code(),
			err,
		)
	}
	if connector == nil {
		return nil, fmt.Errorf(
			"database connector factory %q returned nil connector",
			definition.Connector.Code(),
		)
	}
	if connector.Code() != definition.Connector.Code() {
		return nil, fmt.Errorf(
			"database connector factory %q returned connector %q",
			definition.Connector.Code(),
			connector.Code(),
		)
	}
	if err := connector.Ping(ctx); err != nil {
		return nil, fmt.Errorf(
			"ping database connector %q: %w",
			connector.Code(),
			err,
		)
	}

	a.addProvider("connector:"+string(connector.Code()), connector)

	binding := &bindingRuntime{
		connector: connector,
		adapters:  make(map[kernel.ModuleCode]kernel.ModuleDatabase),
	}
	migrationSourceIDs := make(map[string]struct{})
	seedSourceIDs := make(map[string]struct{})

	for _, factory := range definition.Adapters {
		database, err := factory.Build(connector)
		if err != nil {
			return nil, fmt.Errorf(
				"build database adapter %q on connection %q: %w",
				factory.ModuleCode(),
				connector.Code(),
				err,
			)
		}
		if database == nil {
			return nil, fmt.Errorf(
				"database adapter factory %q returned nil",
				factory.ModuleCode(),
			)
		}
		if database.ModuleCode() != factory.ModuleCode() {
			return nil, fmt.Errorf(
				"database adapter factory %q returned adapter %q",
				factory.ModuleCode(),
				database.ModuleCode(),
			)
		}

		moduleCode := database.ModuleCode()
		binding.adapters[moduleCode] = database
		a.addProvider("database:"+string(moduleCode), database)

		if provider, ok := database.(migrations.Provider); ok {
			plans, err := migrationPlans(
				connector,
				moduleCode,
				provider.MigrationSources(),
				migrationSourceIDs,
			)
			if err != nil {
				return nil, err
			}
			a.migrationPlan = append(a.migrationPlan, plans...)
		}

		if provider, ok := database.(seeds.Provider); ok {
			plans, err := seedPlans(
				connector,
				moduleCode,
				provider.SeedSources(),
				seedSourceIDs,
			)
			if err != nil {
				return nil, err
			}
			a.seedPlan = append(a.seedPlan, plans...)
		}
	}

	return binding, nil
}

func (a *App) collectModuleCommandProviders() {
	for _, profile := range a.definition.Profiles {
		for _, profileModule := range profile.Modules {
			if profileModule.Module == nil {
				continue
			}

			a.addProvider(
				"module:"+string(profileModule.Module.Code()),
				profileModule.Module,
			)
		}
	}
}

func (a *App) addProvider(key string, candidate any) {
	provider, ok := candidate.(console.Provider)
	if !ok {
		return
	}

	for _, registered := range a.providers {
		if providerIdentity(registered) == key {
			return
		}
	}

	a.providers = append(a.providers, keyedProvider{
		key:      key,
		Provider: provider,
	})
}

type keyedProvider struct {
	key string
	console.Provider
}

func providerIdentity(provider console.Provider) string {
	if keyed, ok := provider.(keyedProvider); ok {
		return keyed.key
	}
	return ""
}

type profileResolver map[kernel.ProfileCode]*kernel.ProfileRuntime

func (r profileResolver) ProfileRuntime(
	code kernel.ProfileCode,
) (*kernel.ProfileRuntime, bool) {
	runtime, exists := r[code]
	return runtime, exists
}

func migrationPlans(
	connector kernel.DBConnector,
	moduleCode kernel.ModuleCode,
	sources []migrations.Source,
	used map[string]struct{},
) ([]migrations.Plan, error) {
	if len(sources) == 0 {
		return nil, nil
	}

	target, ok := connector.(migrations.Target)
	if !ok {
		return nil, fmt.Errorf(
			"database connector %q does not support migrations required by module %q",
			connector.Code(),
			moduleCode,
		)
	}

	plans := make([]migrations.Plan, 0, len(sources))
	for _, source := range sources {
		if err := validateSource(connector.Code(), moduleCode, source.ID, used); err != nil {
			return nil, err
		}

		plans = append(plans, migrations.Plan{
			Connection: string(connector.Code()),
			Target:     target,
			Source:     source,
		})
	}

	return plans, nil
}

func seedPlans(
	connector kernel.DBConnector,
	moduleCode kernel.ModuleCode,
	sources []seeds.Source,
	used map[string]struct{},
) ([]seeds.Plan, error) {
	if len(sources) == 0 {
		return nil, nil
	}

	target, ok := connector.(seeds.Target)
	if !ok {
		return nil, fmt.Errorf(
			"database connector %q does not support seeds required by module %q",
			connector.Code(),
			moduleCode,
		)
	}

	plans := make([]seeds.Plan, 0, len(sources))
	for _, source := range sources {
		if err := validateSource(connector.Code(), moduleCode, source.ID, used); err != nil {
			return nil, err
		}

		plans = append(plans, seeds.Plan{
			Connection: string(connector.Code()),
			Target:     target,
			Source:     source,
		})
	}

	return plans, nil
}

func validateSource(
	connectionCode kernel.ConnectionCode,
	moduleCode kernel.ModuleCode,
	sourceID string,
	used map[string]struct{},
) error {
	if sourceID != string(moduleCode) {
		return fmt.Errorf(
			"database binding %q module %q returned source %q",
			connectionCode,
			moduleCode,
			sourceID,
		)
	}
	if _, exists := used[sourceID]; exists {
		return fmt.Errorf(
			"database binding %q contains duplicate source %q",
			connectionCode,
			sourceID,
		)
	}

	used[sourceID] = struct{}{}
	return nil
}

func validateDefinition(definition Definition) error {
	definitions := make(
		[]DatabaseDefinition,
		0,
		len(definition.AdditionalDatabases)+1,
	)
	definitions = append(definitions, definition.MainDatabase)
	definitions = append(definitions, definition.AdditionalDatabases...)

	connectionCodes := make(map[kernel.ConnectionCode]struct{}, len(definitions))
	for bindingIndex, database := range definitions {
		if database.Connector == nil {
			return fmt.Errorf(
				"database definition at index %d has nil connector factory",
				bindingIndex,
			)
		}

		connectionCode := database.Connector.Code()
		if connectionCode == "" {
			return fmt.Errorf(
				"database definition at index %d has empty connection code",
				bindingIndex,
			)
		}
		if _, exists := connectionCodes[connectionCode]; exists {
			return fmt.Errorf(
				"database connection %q is defined more than once",
				connectionCode,
			)
		}
		connectionCodes[connectionCode] = struct{}{}

		moduleCodes := make(map[kernel.ModuleCode]struct{}, len(database.Adapters))
		for adapterIndex, adapter := range database.Adapters {
			if adapter == nil {
				return fmt.Errorf(
					"database definition %q adapter at index %d is nil",
					connectionCode,
					adapterIndex,
				)
			}

			moduleCode := adapter.ModuleCode()
			if moduleCode == "" {
				return fmt.Errorf(
					"database definition %q adapter at index %d has empty module code",
					connectionCode,
					adapterIndex,
				)
			}
			if _, exists := moduleCodes[moduleCode]; exists {
				return fmt.Errorf(
					"database definition %q contains duplicate module %q",
					connectionCode,
					moduleCode,
				)
			}
			moduleCodes[moduleCode] = struct{}{}
		}
	}

	profileCodes := make(map[kernel.ProfileCode]struct{}, len(definition.Profiles))
	for profileIndex, profile := range definition.Profiles {
		if profile.Code == "" {
			return fmt.Errorf("profile at index %d has empty code", profileIndex)
		}
		if _, exists := profileCodes[profile.Code]; exists {
			return fmt.Errorf("profile %q is defined more than once", profile.Code)
		}
		profileCodes[profile.Code] = struct{}{}

		moduleCodes := make(map[kernel.ModuleCode]struct{}, len(profile.Modules))
		for moduleIndex, profileModule := range profile.Modules {
			if profileModule.Module == nil {
				return fmt.Errorf(
					"profile %q module at index %d is nil",
					profile.Code,
					moduleIndex,
				)
			}

			moduleCode := profileModule.Module.Code()
			if moduleCode == "" {
				return fmt.Errorf(
					"profile %q module at index %d has empty code",
					profile.Code,
					moduleIndex,
				)
			}
			if _, exists := moduleCodes[moduleCode]; exists {
				return fmt.Errorf(
					"profile %q contains duplicate module %q",
					profile.Code,
					moduleCode,
				)
			}
			moduleCodes[moduleCode] = struct{}{}
		}
	}

	return nil
}

func cloneDefinition(definition Definition) Definition {
	definition.MainDatabase.Adapters = append(
		[]ModuleDatabaseFactory(nil),
		definition.MainDatabase.Adapters...,
	)
	definition.AdditionalDatabases = append(
		[]DatabaseDefinition(nil),
		definition.AdditionalDatabases...,
	)
	for index := range definition.AdditionalDatabases {
		definition.AdditionalDatabases[index].Adapters = append(
			[]ModuleDatabaseFactory(nil),
			definition.AdditionalDatabases[index].Adapters...,
		)
	}

	definition.Profiles = append([]kernel.Profile(nil), definition.Profiles...)
	for index := range definition.Profiles {
		definition.Profiles[index].Modules = append(
			[]kernel.ProfileModule(nil),
			definition.Profiles[index].Modules...,
		)
	}

	return definition
}

var _ kernel.DatabaseResolver = (*App)(nil)
var _ console.Application = (*App)(nil)
