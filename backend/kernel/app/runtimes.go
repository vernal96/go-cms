package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/console"
	"github.com/vernal96/go-cms/kernel/eventbus"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/security"
)

func (a *App) Definition() Definition {
	if a == nil {
		return Definition{}
	}

	return cloneDefinition(a.definition)
}

func (a *App) Logger() *slog.Logger {
	if a == nil {
		return nil
	}

	return a.logger
}

func (a *App) EventBus() eventbus.Bus {
	if a == nil {
		return nil
	}

	return a.eventBus
}

func (a *App) Console() *console.Console {
	if a == nil {
		return nil
	}

	return a.console
}

func (a *App) ProfileBlueprint(
	code kernel.ProfileCode,
) (*kernel.ProfileBlueprint, bool) {
	if a == nil || a.closed.Load() || !a.booted.Load() {
		return nil, false
	}

	blueprint, exists := a.profileBlueprints[code]
	return blueprint, exists
}

func (a *App) AdminManagement() (*admin.Management, error) {
	if a == nil {
		return nil, errors.New("app is nil")
	}
	if a.closed.Load() {
		return nil, ErrClosed
	}
	if !a.booted.Load() || a.adminManagement == nil {
		return nil, ErrNotBooted
	}
	return a.adminManagement, nil
}

func (a *App) RuntimeByDomain(
	ctx context.Context,
	actor security.Actor,
	domain string,
) (*site.Runtime, error) {
	if a == nil || a.closed.Load() || !a.booted.Load() {
		if a != nil && a.closed.Load() {
			return nil, ErrClosed
		}
		return nil, ErrNotBooted
	}

	return a.sites.ResolveByDomain(ctx, actor, domain)
}

func (a *App) RuntimeBySiteID(
	ctx context.Context,
	actor security.Actor,
	id site.ID,
) (*site.Runtime, error) {
	if a == nil || a.closed.Load() || !a.booted.Load() {
		if a != nil && a.closed.Load() {
			return nil, ErrClosed
		}
		return nil, ErrNotBooted
	}
	runtime, exists := a.sites.RuntimeByID(id)
	if !exists {
		return nil, site.ErrNotFound
	}
	return a.sites.ResolveByDomain(ctx, actor, runtime.Site().Domain)
}

func (a *App) ReloadSites(ctx context.Context) error {
	if a == nil {
		return errors.New("app is nil")
	}
	if ctx == nil {
		return errors.New("site reload context is nil")
	}
	if a.closed.Load() {
		return ErrClosed
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
