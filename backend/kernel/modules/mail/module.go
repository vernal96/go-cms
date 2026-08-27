package mail

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/adminui"
	"github.com/vernal96/go-cms/kernel/background"
	"github.com/vernal96/go-cms/kernel/job"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

const ModuleCode kernel.ModuleCode = "mail"

type Config struct {
	DefaultTransport TransportAlias
	Transports       map[TransportAlias]Transport
	Renderer         RendererConfig
	MessageIDDomain  string
	HistoryRetention time.Duration
	CleanupInterval  time.Duration
	CleanupBatchSize int
}

type Database interface {
	kernel.ModuleDatabase
	Mail() Repository
}

type coreDependency interface {
	kernel.ModuleRuntime
	Files() file.ManagementService
	Authorization() security.Authorizer
}

type Module struct{}

func (Module) Code() kernel.ModuleCode { return ModuleCode }

func (Module) Dependencies() []kernel.ModuleCode { return []kernel.ModuleCode{core.ModuleCode} }

func (Module) ModuleDescriptor() kernel.ModuleDescriptor {
	return kernel.ModuleDescriptor{Label: "Mail", Description: "Шаблоны и асинхронная отправка почты"}
}

func (Module) Registry() kernel.ModuleRegistry {
	return kernel.ModuleRegistry{PermissionEntities: []permission.Entity{
		{Code: "template"},
		{Code: "message", Actions: []permission.Action{permission.Read, permission.Create, permission.Delete}},
	}}
}

func (Module) JobNames() []string { return []string{SendJobName} }

func (Module) BackgroundTaskNames() []string { return []string{"mail.history_retention"} }

func (Module) Build(_ context.Context, ctx kernel.ModuleContext) (kernel.ModuleRuntime, error) {
	database, err := kernel.ModuleDatabaseFrom[Database](ctx, "", ModuleCode)
	if err != nil {
		return nil, err
	}
	if database.Mail() == nil {
		return nil, errors.New("mail repository is nil")
	}
	coreRuntime, err := kernel.ModuleDependencyFrom[coreDependency](ctx, core.ModuleCode)
	if err != nil {
		return nil, err
	}
	config, err := kernel.ModuleConfigFrom[Config](ctx)
	if err != nil {
		return nil, err
	}
	config, err = normalizeConfig(config)
	if err != nil {
		return nil, err
	}
	transports, err := NewTransportRegistry(config.Transports)
	if err != nil {
		return nil, err
	}
	if _, exists := transports.Transport(config.DefaultTransport); !exists {
		return nil, fmt.Errorf("default mail transport %q is unavailable", config.DefaultTransport)
	}
	siteIDValue, err := strconv.ParseInt(ctx.Scope().SiteID(), 10, 64)
	if err != nil || siteIDValue <= 0 {
		return nil, errors.New("mail runtime site scope is invalid")
	}
	renderer, err := NewRenderer(ctx.Registry(), coreRuntime.Files(), ctx.Scope(), config.Renderer)
	if err != nil {
		return nil, err
	}
	service, err := NewService(
		site.ID(siteIDValue), database.Mail(), renderer, coreRuntime.Authorization(),
		transports, config.DefaultTransport, config.MessageIDDomain,
	)
	if err != nil {
		return nil, err
	}
	worker, err := NewWorker(site.ID(siteIDValue), database.Mail(), coreRuntime.Files(), transports)
	if err != nil {
		return nil, err
	}
	return &Runtime{service: service, worker: worker, retention: config.HistoryRetention, cleanupInterval: config.CleanupInterval, cleanupBatchSize: config.CleanupBatchSize}, nil
}

type Runtime struct {
	service          *Service
	worker           *Worker
	retention        time.Duration
	cleanupInterval  time.Duration
	cleanupBatchSize int
}

func (*Runtime) ModuleCode() kernel.ModuleCode { return ModuleCode }
func (r *Runtime) Mail() *Service              { return r.service }

func (r *Runtime) Jobs() []job.Definition {
	return []job.Definition{{Name: SendJobName, ScopeID: fmt.Sprint(r.service.siteID), Handler: r.worker.Handle}}
}

func (r *Runtime) BackgroundTasks() []background.Task {
	if r.retention == 0 {
		return nil
	}
	return []background.Task{{Name: "mail.history_retention", Run: r.runRetention}}
}

func (r *Runtime) runRetention(ctx context.Context) error {
	cleanup := func() error {
		_, err := r.service.repository.Cleanup(ctx, r.retention, r.cleanupBatchSize)
		return err
	}
	if err := cleanup(); err != nil && ctx.Err() == nil {
		return err
	}
	ticker := time.NewTicker(r.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := cleanup(); err != nil && ctx.Err() == nil {
				return err
			}
		}
	}
}

func (r *Runtime) AdminNavigation() []adminui.NavigationItem {
	return []adminui.NavigationItem{{
		Code: "mail", Label: "Почта", Icon: "mail", Order: 50, Scope: adminui.NavigationSite,
		Children: []adminui.NavigationItem{
			{Code: "mail.templates", Label: "Шаблоны", Route: "mail.templates", Order: 10, Permission: TemplateReadPermission, Scope: adminui.NavigationSite},
			{Code: "mail.send", Label: "Отправить", Route: "mail.send", Order: 20, Permission: MessageCreatePermission, Scope: adminui.NavigationSite},
			{Code: "mail.history", Label: "История", Route: "mail.history", Order: 30, Permission: MessageReadPermission, Scope: adminui.NavigationSite},
		},
	}}
}

func normalizeConfig(config Config) (Config, error) {
	if config.DefaultTransport == "" {
		config.DefaultTransport = "default"
	}
	if config.HistoryRetention < 0 || config.CleanupInterval < 0 || config.CleanupBatchSize < 0 {
		return Config{}, errors.New("mail history configuration is invalid")
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = time.Hour
	}
	if config.CleanupBatchSize == 0 {
		config.CleanupBatchSize = 100
	}
	return config, nil
}

var _ kernel.Module = Module{}
var _ kernel.DependencyProvider = Module{}
var _ kernel.ModuleDescriptorProvider = Module{}
var _ kernel.RegistryProvider = Module{}
var _ job.NamesProvider = Module{}
var _ background.NamesProvider = Module{}
var _ kernel.ModuleRuntime = (*Runtime)(nil)
var _ job.Provider = (*Runtime)(nil)
var _ background.Provider = (*Runtime)(nil)
var _ adminui.NavigationProvider = (*Runtime)(nil)
