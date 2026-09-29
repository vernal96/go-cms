package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/vernal96/go-cms-kernel/migrations"
	"github.com/vernal96/go-cms-kernel/seeds"
	"github.com/vernal96/go-cms/internal/config"
)

type fakeApplication struct {
	migrations []migrations.Plan
	seeds      []seeds.Plan
	bootErr    error
	closeErr   error
	calls      []string
}

func (a *fakeApplication) MigrationPlans() []migrations.Plan {
	a.calls = append(a.calls, "migrations")
	return a.migrations
}

func (a *fakeApplication) SeedPlans() []seeds.Plan {
	a.calls = append(a.calls, "seeds")
	return a.seeds
}

func (a *fakeApplication) Boot(context.Context) error {
	a.calls = append(a.calls, "boot")
	return a.bootErr
}

func (a *fakeApplication) Close() error {
	a.calls = append(a.calls, "close")
	return a.closeErr
}

func TestPrepareFailuresCloseApplication(t *testing.T) {
	bootErr := errors.New("boot failed")
	closeErr := errors.New("close failed")
	for _, test := range []struct {
		name  string
		app   fakeApplication
		calls []string
	}{
		{"migrations", fakeApplication{migrations: []migrations.Plan{{}}}, []string{"migrations", "close"}},
		{"seeds", fakeApplication{seeds: []seeds.Plan{{Source: seeds.Source{Tags: []seeds.Tag{"prod"}}}}}, []string{"migrations", "seeds", "close"}},
		{"boot", fakeApplication{bootErr: bootErr}, []string{"migrations", "seeds", "boot", "close"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.app.closeErr = closeErr
			err := prepare(context.Background(), &test.app, HTTP)
			if err == nil || !errors.Is(err, closeErr) || !strings.Contains(err.Error(), test.name) {
				t.Fatalf("startup and close errors must both survive: %v", err)
			}
			if test.name == "boot" && !errors.Is(err, bootErr) {
				t.Fatal("boot error identity was lost")
			}
			if !reflect.DeepEqual(test.app.calls, test.calls) {
				t.Fatalf("calls = %v, want %v", test.app.calls, test.calls)
			}
		})
	}
}

func TestPrepareHTTPTransfersOwnership(t *testing.T) {
	// An invalid dev-only seed must not be executed during ordinary startup.
	a := &fakeApplication{seeds: []seeds.Plan{{Source: seeds.Source{Tags: []seeds.Tag{"dev"}}}}}
	if err := prepare(context.Background(), a, HTTP); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.calls, []string{"migrations", "seeds", "boot"}) {
		t.Fatalf("unexpected lifecycle: %v", a.calls)
	}
}

func TestProductionSeedsIncludesSharedButNotDevOnlyData(t *testing.T) {
	plans := []seeds.Plan{
		{Source: seeds.Source{ID: "starter", Tags: []seeds.Tag{"dev"}}},
		{Source: seeds.Source{ID: "identity_shared", Tags: []seeds.Tag{"dev", "prod"}}},
		{Source: seeds.Source{ID: "project_prod", Tags: []seeds.Tag{"prod"}}},
	}
	if selected := productionSeeds(plans); !reflect.DeepEqual(selected, plans[1:]) {
		t.Fatal("startup must apply prod and shared seeds, leaving demo data for manual execution")
	}
}

func TestPrepareConsoleDoesNotInitialize(t *testing.T) {
	a := &fakeApplication{
		migrations: []migrations.Plan{{}}, seeds: []seeds.Plan{{}},
		bootErr: errors.New("must not boot"),
	}
	if err := prepare(context.Background(), a, Console); err != nil {
		t.Fatal(err)
	}
	if len(a.calls) != 0 {
		t.Fatalf("console must leave migrations, seeds and Boot to commands: %v", a.calls)
	}
}

func TestPrepareCancellationClosesApplication(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a := &fakeApplication{}
	if err := prepare(ctx, a, Console); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if !reflect.DeepEqual(a.calls, []string{"close"}) {
		t.Fatalf("unexpected lifecycle: %v", a.calls)
	}
}

func TestNewRejectsInvalidModeAndCanceledContext(t *testing.T) {
	if a, err := New(context.Background(), config.Config{}, nil, Mode(-1)); err == nil || a != nil {
		t.Fatal("invalid mode must fail before opening infrastructure")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if a, err := New(ctx, config.Config{}, nil, HTTP); !errors.Is(err, context.Canceled) || a != nil {
		t.Fatalf("canceled startup: app=%v, err=%v", a, err)
	}
}
