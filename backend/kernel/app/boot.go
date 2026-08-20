package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	"github.com/vernal96/go-cms/kernel/modules/core"
	coregroup "github.com/vernal96/go-cms/kernel/modules/core/group"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	coreuser "github.com/vernal96/go-cms/kernel/modules/core/user"
)

func (a *App) Boot(ctx context.Context) error {
	if a == nil {
		return errors.New("app is nil")
	}

	a.bootOnce.Do(func() {
		logContext := ctx
		if logContext == nil {
			logContext = context.Background()
		}
		if a.logger != nil {
			a.logger.InfoContext(
				logContext,
				"application boot started",
				slog.String("event", "app.boot.started"),
			)
		}
		a.bootErr = a.boot(ctx)
		if a.logger == nil {
			return
		}
		if a.bootErr != nil {
			a.logger.ErrorContext(
				logContext,
				"application boot failed",
				slog.String("event", "app.boot.failed"),
				slog.Any("error", a.bootErr),
			)
			return
		}
		a.logger.InfoContext(
			logContext,
			"application boot completed",
			slog.String("event", "app.boot.completed"),
		)
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

	coreServices, err := core.NewServices(
		a.coreDatabase,
		a.permissions,
		a.filesystems,
		a.definition.PasswordHasher,
		a.caches,
	)
	if err != nil {
		return err
	}
	a.coreDatabase = coreServices.Database()
	a.main.adapters[core.ModuleCode] = a.coreDatabase
	profiles, err := bindCoreServices(a.definition.Profiles, coreServices)
	if err != nil {
		return err
	}

	factory, err := kernel.NewProfileRuntimeFactory(
		a,
		kernel.RuntimeServices{
			Caches:      a.caches,
			Filesystems: a.filesystems,
			EventBus:    a.eventBus,
			Logger:      a.logger,
		},
	)
	if err != nil {
		return err
	}

	profileBlueprints := make(
		map[kernel.ProfileCode]*kernel.ProfileBlueprint,
		len(a.definition.Profiles),
	)

	for _, profile := range profiles {
		blueprint, err := factory.Compile(ctx, profile)
		if err != nil {
			return fmt.Errorf(
				"compile profile blueprint %q: %w",
				profile.Code,
				err,
			)
		}

		profileBlueprints[profile.Code] = blueprint
	}

	if err := coreServices.BuildContent(
		ctx,
		profileResolver(profileBlueprints),
	); err != nil {
		return err
	}
	catalog := coreServices.Sites
	resourceService := coreServices.Resources
	siteManagementRepository, ok := a.coreDatabase.Sites().(site.ManagementRepository)
	if !ok {
		return errors.New("site management repository is unavailable")
	}
	resourceManagementRepository, ok := a.coreDatabase.Resources().(resource.ManagementRepository)
	if !ok {
		return errors.New("resource management repository is unavailable")
	}
	userManagementRepository, ok := a.coreDatabase.Users().(coreuser.ManagementRepository)
	if !ok {
		return errors.New("user management repository is unavailable")
	}
	groupManagementRepository, ok := a.coreDatabase.Groups().(coregroup.ManagementRepository)
	if !ok {
		return errors.New("group management repository is unavailable")
	}

	adminManagement, err := admin.NewManagement(admin.ManagementDependencies{
		Profiles:           a.definition.Profiles,
		ProfileSource:      profileResolver(profileBlueprints),
		SiteRepository:     siteManagementRepository,
		Sites:              catalog,
		Resources:          resourceService,
		ResourceRepository: resourceManagementRepository,
		Authorizer:         coreServices.Authorization,
		Permissions:        a.permissions,
		SiteAccessPolicy:   a.definition.SiteAccessPolicy,
		Users:              coreServices.Users,
		UserRepository:     userManagementRepository,
		Groups:             coreServices.Groups,
		GroupRepository:    groupManagementRepository,
		Access:             coreServices.Authorization,
		Files:              coreServices.Files,
		Media:              coreServices.Media,
		MaxUploadSize:      a.definition.MaxUploadSize,
		UploadTimeout:      a.definition.UploadTimeout,
		AvatarStorage:      a.definition.AvatarStorage,
		AvatarMaxSize:      a.definition.AvatarMaxSize,
	})
	if err != nil {
		return err
	}

	a.profileBlueprints = profileBlueprints
	a.sites = catalog
	a.services = servicesFromCore(coreServices)
	a.adminManagement = adminManagement
	a.booted.Store(true)
	return nil
}
