package bootstrap

import (
	"context"
	"errors"

	"github.com/vernal96/go-cms/app"
	"github.com/vernal96/go-cms/internal/config"
	"github.com/vernal96/go-cms/internal/profiles/dev"
	"github.com/vernal96/go-cms/kernel"
)

func NewApp(
	ctx context.Context,
	projectConfig *config.Config,
) (*app.App, error) {
	infrastructure, err := NewInfrastructure(ctx, projectConfig)
	if err != nil {
		return nil, err
	}

	runtime, err := app.New(
		ctx,
		infrastructure.AppConfig(),
		Profiles(),
	)
	if err != nil {
		return nil, errors.Join(err, infrastructure.Close())
	}

	infrastructure.Release()
	return runtime, nil
}

func Profiles() []kernel.Profile {
	return []kernel.Profile{dev.Profile}
}
