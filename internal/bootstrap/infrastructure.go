package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/vernal96/go-cms/app"
	"github.com/vernal96/go-cms/internal/config"
	"github.com/vernal96/go-cms/internal/connectors/mainpostgres"
	"github.com/vernal96/go-cms/kernel"
	corepostgres "github.com/vernal96/go-cms/kernel/modules/core/adapters/postgres"
)

type Infrastructure struct {
	config     app.AppConfig
	connectors []kernel.DBConnector

	released  atomic.Bool
	closeOnce sync.Once
	closeErr  error
}

func NewInfrastructure(
	ctx context.Context,
	projectConfig *config.Config,
) (_ *Infrastructure, resultErr error) {
	if projectConfig == nil {
		return nil, errors.New("project config is nil")
	}

	mainConnector, err := mainpostgres.New(ctx, projectConfig.Postgres)
	if err != nil {
		return nil, err
	}

	infrastructure := &Infrastructure{
		connectors: []kernel.DBConnector{mainConnector},
	}

	defer func() {
		if resultErr == nil {
			return
		}

		resultErr = errors.Join(resultErr, infrastructure.Close())
	}()

	coreDatabase, err := corepostgres.NewDatabase(mainConnector)
	if err != nil {
		return nil, err
	}

	infrastructure.config = app.AppConfig{
		MainDatabase: app.DatabaseBinding{
			Connector: mainConnector,
			Adapters: []kernel.ModuleDatabase{
				coreDatabase,
			},
		},
	}

	return infrastructure, nil
}

func (i *Infrastructure) AppConfig() app.AppConfig {
	if i == nil {
		return app.AppConfig{}
	}

	config := i.config
	config.MainDatabase.Adapters = append(
		[]kernel.ModuleDatabase(nil),
		config.MainDatabase.Adapters...,
	)
	config.AdditionalDatabases = append(
		[]app.DatabaseBinding(nil),
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

func (i *Infrastructure) Release() {
	if i != nil {
		i.released.Store(true)
	}
}

func (i *Infrastructure) Close() error {
	if i == nil || i.released.Load() {
		return nil
	}

	i.closeOnce.Do(func() {
		var closeErrors []error

		for index := len(i.connectors) - 1; index >= 0; index-- {
			connector := i.connectors[index]
			if err := connector.Close(); err != nil {
				closeErrors = append(closeErrors, fmt.Errorf(
					"close database connector %q: %w",
					connector.Code(),
					err,
				))
			}
		}

		i.closeErr = errors.Join(closeErrors...)
	})

	return i.closeErr
}
