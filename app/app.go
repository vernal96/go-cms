package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

var ErrClosed = errors.New("app is closed")

type DatabaseBinding struct {
	Connector kernel.DBConnector
	Adapters  []kernel.ModuleDatabase
}

type AppConfig struct {
	MainDatabase        DatabaseBinding
	AdditionalDatabases []DatabaseBinding
}

type bindingRuntime struct {
	connector kernel.DBConnector
	adapters  map[kernel.ModuleCode]kernel.ModuleDatabase
}

type compiledConfig struct {
	config       AppConfig
	main         *bindingRuntime
	additional   map[kernel.ConnectionCode]*bindingRuntime
	connectors   []kernel.DBConnector
	migrations   []migrations.Plan
	coreDatabase core.Database
}

type App struct {
	config AppConfig

	main       *bindingRuntime
	additional map[kernel.ConnectionCode]*bindingRuntime
	connectors []kernel.DBConnector
	plans      []migrations.Plan

	profileRuntimes map[kernel.ProfileCode]*kernel.ProfileRuntime
	sites           *site.Catalog

	closed      atomic.Bool
	lifecycleMu sync.RWMutex
	closeOnce   sync.Once
	closeErr    error
}

func New(
	ctx context.Context,
	config AppConfig,
	profiles []kernel.Profile,
) (_ *App, resultErr error) {
	if ctx == nil {
		return nil, errors.New("app context is nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	compiled, err := compileConfig(config)
	if err != nil {
		return nil, err
	}

	instance := &App{
		config:          compiled.config,
		main:            compiled.main,
		additional:      compiled.additional,
		connectors:      compiled.connectors,
		plans:           compiled.migrations,
		profileRuntimes: make(map[kernel.ProfileCode]*kernel.ProfileRuntime),
	}

	// From this point connector ownership belongs to App, including failures
	// during ping, migration or runtime compilation.
	defer func() {
		if resultErr == nil {
			return
		}

		resultErr = errors.Join(resultErr, instance.Close())
	}()

	for _, connector := range instance.connectors {
		if err := connector.Ping(ctx); err != nil {
			return nil, fmt.Errorf(
				"ping database connector %q: %w",
				connector.Code(),
				err,
			)
		}
	}

	manager := migrations.NewManager()
	if err := manager.UpAll(ctx, instance.plans); err != nil {
		return nil, err
	}

	factory, err := kernel.NewProfileRuntimeFactory(instance)
	if err != nil {
		return nil, err
	}

	for index, profile := range profiles {
		if profile.Code == "" {
			return nil, fmt.Errorf("profile at index %d has empty code", index)
		}

		if _, exists := instance.profileRuntimes[profile.Code]; exists {
			return nil, fmt.Errorf(
				"profile %q is registered more than once",
				profile.Code,
			)
		}

		runtime, err := factory.Make(ctx, profile)
		if err != nil {
			return nil, fmt.Errorf(
				"compile profile runtime %q: %w",
				profile.Code,
				err,
			)
		}

		if _, exists := runtime.Registry().Module(core.ModuleCode); !exists {
			return nil, fmt.Errorf(
				"profile %q does not contain required module %q",
				profile.Code,
				core.ModuleCode,
			)
		}

		instance.profileRuntimes[profile.Code] = runtime
	}

	catalog, err := site.NewCatalog(
		compiled.coreDatabase.Sites(),
		instance,
	)
	if err != nil {
		return nil, err
	}

	instance.sites = catalog

	if err := catalog.Reload(ctx); err != nil {
		return nil, fmt.Errorf("compile site runtimes: %w", err)
	}

	return instance, nil
}

func (a *App) Config() AppConfig {
	if a == nil {
		return AppConfig{}
	}

	return cloneConfig(a.config)
}

func (a *App) ProfileRuntime(
	code kernel.ProfileCode,
) (*kernel.ProfileRuntime, bool) {
	if a == nil || a.closed.Load() {
		return nil, false
	}

	runtime, exists := a.profileRuntimes[code]
	return runtime, exists
}

func (a *App) RuntimeByDomain(
	domain string,
) (*site.Runtime, bool) {
	if a == nil || a.closed.Load() || a.sites == nil {
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

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()

	if a.closed.Load() {
		return ErrClosed
	}

	if err := a.sites.Reload(ctx); err != nil {
		return err
	}

	return nil
}

func (a *App) MigrationPlans() []migrations.Plan {
	if a == nil {
		return nil
	}

	return append([]migrations.Plan(nil), a.plans...)
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

// MigrationPlans validates database bindings without opening runtimes and
// returns plans in binding/adapter/source declaration order.
func MigrationPlans(config AppConfig) ([]migrations.Plan, error) {
	compiled, err := compileConfig(config)
	if err != nil {
		return nil, err
	}

	return append([]migrations.Plan(nil), compiled.migrations...), nil
}

func compileConfig(config AppConfig) (*compiledConfig, error) {
	config = cloneConfig(config)

	inputs := make([]DatabaseBinding, 0, len(config.AdditionalDatabases)+1)
	inputs = append(inputs, config.MainDatabase)
	inputs = append(inputs, config.AdditionalDatabases...)

	compiled := &compiledConfig{
		config:     config,
		additional: make(map[kernel.ConnectionCode]*bindingRuntime),
		connectors: make([]kernel.DBConnector, 0, len(inputs)),
	}

	connectionCodes := make(map[kernel.ConnectionCode]struct{}, len(inputs))

	for bindingIndex, input := range inputs {
		if input.Connector == nil {
			return nil, fmt.Errorf(
				"database binding at index %d has nil connector",
				bindingIndex,
			)
		}

		connectionCode := input.Connector.Code()
		if connectionCode == "" {
			return nil, fmt.Errorf(
				"database binding at index %d has empty connection code",
				bindingIndex,
			)
		}

		if _, exists := connectionCodes[connectionCode]; exists {
			return nil, fmt.Errorf(
				"database connection %q is registered more than once",
				connectionCode,
			)
		}
		connectionCodes[connectionCode] = struct{}{}

		binding := &bindingRuntime{
			connector: input.Connector,
			adapters:  make(map[kernel.ModuleCode]kernel.ModuleDatabase),
		}

		sourceIDs := make(map[string]struct{})

		for adapterIndex, adapter := range input.Adapters {
			if adapter == nil {
				return nil, fmt.Errorf(
					"database binding %q adapter at index %d is nil",
					connectionCode,
					adapterIndex,
				)
			}

			moduleCode := adapter.ModuleCode()
			if moduleCode == "" {
				return nil, fmt.Errorf(
					"database binding %q adapter at index %d has empty module code",
					connectionCode,
					adapterIndex,
				)
			}

			if _, exists := binding.adapters[moduleCode]; exists {
				return nil, fmt.Errorf(
					"database binding %q contains duplicate module %q",
					connectionCode,
					moduleCode,
				)
			}
			binding.adapters[moduleCode] = adapter

			provider, hasMigrations := adapter.(migrations.Provider)
			if !hasMigrations {
				continue
			}

			sources := provider.MigrationSources()
			if len(sources) == 0 {
				continue
			}

			target, ok := input.Connector.(migrations.Target)
			if !ok {
				return nil, fmt.Errorf(
					"database connector %q does not support migrations required by module %q",
					connectionCode,
					moduleCode,
				)
			}

			for _, source := range sources {
				if source.ID != string(moduleCode) {
					return nil, fmt.Errorf(
						"database binding %q module %q returned migration source %q",
						connectionCode,
						moduleCode,
						source.ID,
					)
				}

				if _, exists := sourceIDs[source.ID]; exists {
					return nil, fmt.Errorf(
						"database binding %q contains duplicate migration source %q",
						connectionCode,
						source.ID,
					)
				}
				sourceIDs[source.ID] = struct{}{}

				compiled.migrations = append(compiled.migrations, migrations.Plan{
					Connection: string(connectionCode),
					Target:     target,
					Source:     source,
				})
			}
		}

		compiled.connectors = append(compiled.connectors, input.Connector)

		if bindingIndex == 0 {
			compiled.main = binding
		} else {
			compiled.additional[connectionCode] = binding
		}
	}

	coreAdapter, exists := compiled.main.adapters[core.ModuleCode]
	if !exists {
		return nil, errors.New("main database binding does not contain core database")
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

	compiled.coreDatabase = coreDatabase
	return compiled, nil
}

func cloneConfig(config AppConfig) AppConfig {
	config.MainDatabase.Adapters = append(
		[]kernel.ModuleDatabase(nil),
		config.MainDatabase.Adapters...,
	)

	config.AdditionalDatabases = append(
		[]DatabaseBinding(nil),
		config.AdditionalDatabases...,
	)

	for index := range config.AdditionalDatabases {
		config.AdditionalDatabases[index].Adapters = append(
			[]kernel.ModuleDatabase(nil),
			config.AdditionalDatabases[index].Adapters...,
		)
	}

	return config
}

var _ kernel.DatabaseResolver = (*App)(nil)
var _ site.ProfileResolver = (*App)(nil)
