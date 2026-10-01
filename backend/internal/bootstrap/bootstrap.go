// Package bootstrap assembles this project's application using its declarations.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"slices"

	kernel "github.com/vernal96/go-cms-kernel"
	"github.com/vernal96/go-cms-kernel/app"
	"github.com/vernal96/go-cms-kernel/migrations"
	"github.com/vernal96/go-cms-kernel/seeds"
	"github.com/vernal96/go-cms/internal/config"
	"github.com/vernal96/go-cms/internal/profiles/starter"
	"github.com/vernal96/go-cms/internal/settings"
)

type Mode int

const (
	HTTP Mode = iota
	Console
)

// New transfers ownership of the application to the caller on success.
// HTTP applies migrations and prod-tagged seeds before Boot. Console leaves initialization
// to kernel commands, allowing schema commands to run before Boot.
func New(ctx context.Context, cfg config.Config, seedFiles fs.FS, mode Mode) (*app.App, error) {
	if mode != HTTP && mode != Console {
		return nil, fmt.Errorf("unknown bootstrap mode %d", mode)
	}
	definition := settings.Config{
		LoggerPath:     cfg.LoggerPath,
		Infrastructure: cfg.Infrastructure.Definition(),
		Profiles:       []kernel.Profile{starter.Profile},
		SeedFiles:      seedFiles,
	}.Definition()
	application, err := app.New(ctx, definition)
	if err != nil {
		// app.New owns cleanup of partially opened infrastructure.
		return nil, err
	}
	slog.SetDefault(application.Logger())
	if err := prepare(ctx, application, mode); err != nil {
		return nil, err
	}
	return application, nil
}

type lifecycle interface {
	MigrationPlans() []migrations.Plan
	SeedPlans() []seeds.Plan
	Boot(context.Context) error
	Close() error
}

func prepare(ctx context.Context, application lifecycle, mode Mode) (resultErr error) {
	defer func() {
		if resultErr != nil {
			resultErr = errors.Join(resultErr, application.Close())
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	if mode == Console {
		return nil
	}
	if err := migrations.NewManager().UpAll(ctx, application.MigrationPlans()); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	if err := seeds.NewManager().UpAll(ctx, productionSeeds(application.SeedPlans())); err != nil {
		return fmt.Errorf("apply seeds: %w", err)
	}
	if err := application.Boot(ctx); err != nil {
		return fmt.Errorf("boot application: %w", err)
	}
	return nil
}

// Shared module seeds may have both dev and prod tags. Dev-only project data
// remains available to the console but is never applied by HTTP startup.
func productionSeeds(plans []seeds.Plan) []seeds.Plan {
	var selected []seeds.Plan
	for _, plan := range plans {
		if slices.Contains(plan.Source.Tags, seeds.Tag("prod")) {
			selected = append(selected, plan)
		}
	}
	return selected
}
