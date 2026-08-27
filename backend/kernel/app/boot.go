package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sort"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/background"
	"github.com/vernal96/go-cms/kernel/job"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	"github.com/vernal96/go-cms/kernel/modules/core"
	coregroup "github.com/vernal96/go-cms/kernel/modules/core/group"
	coremanagement "github.com/vernal96/go-cms/kernel/modules/core/management"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	coreuser "github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/outbox"
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
	siteAccessPolicy := a.definition.SiteAccessPolicy
	if siteAccessPolicy == nil {
		siteAccessPolicy, err = coremanagement.NewGroupSiteAccessPolicy(
			groupManagementRepository,
			coreServices.Authorization,
		)
		if err != nil {
			return err
		}
	}
	cmsSites, err := coremanagement.NewSites(coremanagement.SiteDependencies{
		Profiles:         a.definition.Profiles,
		ProfileSource:    profileResolver(profileBlueprints),
		SiteRepository:   siteManagementRepository,
		Sites:            catalog,
		Resources:        resourceService,
		Authorizer:       coreServices.Authorization,
		SiteAccessPolicy: siteAccessPolicy,
	})
	if err != nil {
		return err
	}
	cmsResources, err := coremanagement.NewResources(coremanagement.ResourceDependencies{
		Sites: catalog, Resources: resourceService, LibraryItems: coreServices.LibraryItems,
		Revisions:          coreServices.Revisions,
		Administrator:      coreServices.Authorization,
		ResourceRepository: resourceManagementRepository,
		Authorizer:         coreServices.Authorization, SiteAccessPolicy: siteAccessPolicy,
	})
	if err != nil {
		return err
	}
	cmsFiles, err := coremanagement.NewFiles(coreServices.Files, coreServices.Authorization)
	if err != nil {
		return err
	}

	adminManagement, err := admin.NewManagement(admin.ManagementDependencies{
		Profiles:           a.definition.Profiles,
		SiteRepository:     siteManagementRepository,
		Sites:              catalog,
		ResourceRepository: resourceManagementRepository,
		Authorizer:         coreServices.Authorization,
		Permissions:        a.permissions,
		SiteAccessPolicy:   siteAccessPolicy,
		Users:              coreServices.Users,
		UserRepository:     userManagementRepository,
		Groups:             coreServices.Groups,
		GroupRepository:    groupManagementRepository,
		Access:             coreServices.Authorization,
		Files:              coreServices.Files,
		Media:              coreServices.Media,
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
	a.cmsSites = cmsSites
	a.cmsResources = cmsResources
	a.cmsFiles = cmsFiles
	a.adminManagement = adminManagement
	a.siteAccessPolicy = siteAccessPolicy
	jobRunner, err := jobRunnerFromProfiles(a.definition.Profiles, catalog)
	if err != nil {
		return err
	}
	backgroundTasks, err := backgroundTasksFromProfiles(a.definition.Profiles, catalog)
	if err != nil {
		return err
	}
	var publisher *outbox.Publisher
	if len(a.outboxSources) > 0 {
		publisher, err = outbox.NewPublisher(a.eventBus, a.outboxSources, a.logger, a.definition.OutboxPublisher)
		if err != nil {
			return err
		}
	}
	var workerContext context.Context
	if len(a.outboxSources) > 0 || jobRunner != nil || len(backgroundTasks) > 0 {
		var cancel context.CancelFunc
		workerContext, cancel = context.WithCancel(context.Background())
		a.workerCancel = cancel
	}
	if len(a.outboxSources) > 0 {
		a.outboxPublisher = publisher
		a.workers.Add(1)
		go func() {
			defer a.workers.Done()
			if err := publisher.Run(workerContext); err != nil && a.logger != nil {
				a.logger.Error("outbox publisher exited", slog.String("event", "outbox.publisher.failed"), slog.Any("error", err))
			}
		}()
	}
	if jobRunner != nil {
		a.workers.Add(1)
		go func() {
			defer a.workers.Done()
			if err := jobRunner.Run(workerContext, a.eventBus, "go-cms"); err != nil && a.logger != nil && workerContext.Err() == nil {
				a.logger.Error("job runner exited", slog.String("event", "job.runner.failed"), slog.Any("error", err))
			}
		}()
	}
	for _, task := range backgroundTasks {
		task := task
		a.workers.Add(1)
		go func() {
			defer a.workers.Done()
			for workerContext.Err() == nil {
				err := task.Run(workerContext)
				if workerContext.Err() != nil {
					return
				}
				if err != nil && a.logger != nil {
					a.logger.Error("background task exited", slog.String("event", "background.task.failed"), slog.String("task", task.Name), slog.Any("error", err))
				}
				timer := time.NewTimer(5 * time.Second)
				select {
				case <-workerContext.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
			}
		}()
	}
	a.booted.Store(true)
	return nil
}

func backgroundTasksFromProfiles(profiles []kernel.Profile, catalog *site.Catalog) ([]background.Task, error) {
	names, err := declaredNames(profiles, func(module kernel.Module) ([]string, bool) {
		provider, ok := module.(background.NamesProvider)
		if !ok {
			return nil, false
		}
		return provider.BackgroundTaskNames(), true
	})
	if err != nil {
		return nil, fmt.Errorf("collect background task declarations: %w", err)
	}
	result := make([]background.Task, len(names))
	for index, name := range names {
		name := name
		result[index] = background.Task{Name: name, Run: func(ctx context.Context) error {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				if task, exists := runtimeBackgroundTask(catalog, name); exists {
					return task.Run(ctx)
				}
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
				}
			}
		}}
	}
	return result, nil
}

func runtimeBackgroundTask(catalog *site.Catalog, name string) (background.Task, bool) {
	for _, runtime := range catalog.Runtimes() {
		for _, moduleRuntime := range runtime.Profile().Modules() {
			provider, ok := moduleRuntime.(background.Provider)
			if !ok {
				continue
			}
			for _, task := range provider.BackgroundTasks() {
				if task.Name == name && task.Run != nil {
					return task, true
				}
			}
		}
	}
	return background.Task{}, false
}

func jobRunnerFromProfiles(profiles []kernel.Profile, catalog *site.Catalog) (*job.Runner, error) {
	names, err := declaredNames(profiles, func(module kernel.Module) ([]string, bool) {
		provider, ok := module.(job.NamesProvider)
		if !ok {
			return nil, false
		}
		return provider.JobNames(), true
	})
	if err != nil {
		return nil, fmt.Errorf("collect job declarations: %w", err)
	}
	registry := job.NewRegistry()
	if len(names) == 0 {
		return nil, nil
	}
	for _, name := range names {
		name := name
		if err := registry.Register(name, func(ctx context.Context, item job.Envelope) error {
			candidates := make([]job.Definition, 0, 1)
			for _, runtime := range catalog.Runtimes() {
				if item.ScopeID != "" && item.ScopeID != fmt.Sprint(runtime.Site().ID) {
					continue
				}
				for _, moduleRuntime := range runtime.Profile().Modules() {
					provider, ok := moduleRuntime.(job.Provider)
					if !ok {
						continue
					}
					for _, definition := range provider.Jobs() {
						if definition.Name != name || definition.Handler == nil || (definition.ScopeID != "" && definition.ScopeID != item.ScopeID) {
							continue
						}
						candidates = append(candidates, definition)
					}
				}
			}
			if len(candidates) == 0 {
				return fmt.Errorf("job handler %q scope %q has no current site runtime", name, item.ScopeID)
			}
			if len(candidates) > 1 {
				return fmt.Errorf("job handler %q scope %q is ambiguous", name, item.ScopeID)
			}
			return candidates[0].Handler(ctx, item)
		}); err != nil {
			return nil, err
		}
	}
	return job.NewRunner(registry)
}

func declaredNames(
	profiles []kernel.Profile,
	resolve func(kernel.Module) ([]string, bool),
) ([]string, error) {
	owners := make(map[string]kernel.ModuleCode)
	modules := make(map[kernel.ModuleCode][]string)
	for _, profile := range profiles {
		for _, profileModule := range profile.Modules {
			names, ok := resolve(profileModule.Module)
			if !ok {
				continue
			}
			names = append([]string(nil), names...)
			sort.Strings(names)
			if previous, exists := modules[profileModule.Module.Code()]; exists {
				if !slices.Equal(previous, names) {
					return nil, fmt.Errorf("module %q has inconsistent declarations", profileModule.Module.Code())
				}
				continue
			}
			modules[profileModule.Module.Code()] = names
			for _, name := range names {
				if name == "" {
					return nil, fmt.Errorf("module %q declares an empty name", profileModule.Module.Code())
				}
				if owner, exists := owners[name]; exists && owner != profileModule.Module.Code() {
					return nil, fmt.Errorf("name %q is declared by modules %q and %q", name, owner, profileModule.Module.Code())
				}
				owners[name] = profileModule.Module.Code()
			}
		}
	}
	result := make([]string, 0, len(owners))
	for name := range owners {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}
