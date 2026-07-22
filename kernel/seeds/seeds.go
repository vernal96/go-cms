package seeds

import (
	"context"

	"github.com/vernal96/go-cms/kernel/migrations"
)

const DefaultHistoryTable = "schema_seeds"

type Source = migrations.Source
type Target = migrations.Target
type Plan = migrations.Plan
type MigrationStatus = migrations.MigrationStatus
type Status = migrations.Status

type Provider interface {
	SeedSources() []Source
}

type Manager struct {
	manager *migrations.Manager
}

func NewManager() *Manager {
	return &Manager{
		manager: migrations.NewManagerWithHistoryTable(
			DefaultHistoryTable,
		),
	}
}

func (m *Manager) Up(ctx context.Context, plan Plan) error {
	return m.manager.Up(ctx, plan)
}

func (m *Manager) UpAll(ctx context.Context, plans []Plan) error {
	return m.manager.UpAll(ctx, plans)
}

func (m *Manager) Down(
	ctx context.Context,
	plan Plan,
	steps int,
) error {
	return m.manager.Down(ctx, plan, steps)
}

func (m *Manager) DownAll(
	ctx context.Context,
	plans []Plan,
	steps int,
) error {
	return m.manager.DownAll(ctx, plans, steps)
}

func (m *Manager) Version(
	ctx context.Context,
	plan Plan,
) (uint, bool, bool, error) {
	return m.manager.Version(ctx, plan)
}

func (m *Manager) Force(
	ctx context.Context,
	plan Plan,
	version int,
) error {
	return m.manager.Force(ctx, plan, version)
}

func (m *Manager) Status(
	ctx context.Context,
	plan Plan,
) (Status, error) {
	return m.manager.Status(ctx, plan)
}

func (m *Manager) StatusAll(
	ctx context.Context,
	plans []Plan,
) ([]Status, error) {
	return m.manager.StatusAll(ctx, plans)
}
