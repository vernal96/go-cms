package mail

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/adminui"
	"github.com/vernal96/go-cms/kernel/background"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/job"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
	httptransport "github.com/vernal96/go-cms/kernel/transport/http"
)

const ModuleCode kernel.ModuleCode = "mail"

type Config struct {
	DefaultTransport     TransportAlias
	Transports           map[TransportAlias]Transport
	Renderer             RendererConfig
	MessageIDDomain      string
	HistoryRetention     time.Duration
	CleanupInterval      time.Duration
	CleanupBatchSize     int
	SendMaxAttempts      int
	MaxRecipients        int
	MaxMessageSize       int64
	MaxAttachmentSize    int64
	UploadStorage        filesystem.Code
	UploadPath           string
	SpoolEnabled         bool
	SpoolTTL             time.Duration
	SpoolCleanupInterval time.Duration
	SpoolCleanupBatch    int
}

type Database interface {
	kernel.ModuleDatabase
	Mail() Repository
}

type coreDependency interface {
	kernel.ModuleRuntime
	Files() file.ManagementService
	Authorization() security.Authorizer
	Users() user.Service
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
	var spool *AttachmentSpool
	if config.SpoolEnabled {
		disk, exists := ctx.Filesystems().Disk(SpoolFilesystemAlias)
		if !exists {
			return nil, errors.New("mail spool filesystem binding is unavailable")
		}
		spool, err = NewAttachmentSpool(site.ID(siteIDValue), disk)
		if err != nil {
			return nil, err
		}
	}
	scope := ctx.Scope()
	renderer, err := NewRenderer(ctx.Registry(), coreRuntime.Files(), site.Site{
		ID: site.ID(siteIDValue), ProfileCode: ctx.Profile().Code, Domain: scope.Domain(),
		Locale: scope.Locale(), IsPublic: scope.IsPublic(), Settings: scope.Settings(),
	}, ctx.Profile().Params, config.Renderer)
	if err != nil {
		return nil, err
	}
	service, err := NewService(
		site.ID(siteIDValue), database.Mail(), renderer, coreRuntime.Authorization(),
		coreRuntime.Users(), transports, spool, Limits{MaxRecipients: config.MaxRecipients, MaxMessageSize: config.MaxMessageSize, MaxAttachmentSize: config.MaxAttachmentSize}, config.DefaultTransport, config.MessageIDDomain,
	)
	if err != nil {
		return nil, err
	}
	service.logger = ctx.Logger()
	service.uploadStorage = config.UploadStorage
	service.uploadPath = config.UploadPath
	worker, err := NewWorker(site.ID(siteIDValue), database.Mail(), coreRuntime.Files(), spool, transports, config.SendMaxAttempts, ctx.Logger())
	if err != nil {
		return nil, err
	}
	handler, err := NewHTTPHandler(service)
	if err != nil {
		return nil, err
	}
	return &Runtime{service: service, worker: worker, managementHTTP: handler, spool: spool, logger: ctx.Logger(), retention: config.HistoryRetention, cleanupInterval: config.CleanupInterval, cleanupBatchSize: config.CleanupBatchSize, spoolTTL: config.SpoolTTL, spoolCleanupInterval: config.SpoolCleanupInterval, spoolCleanupBatch: config.SpoolCleanupBatch}, nil
}

type Runtime struct {
	service              *Service
	worker               *Worker
	managementHTTP       http.Handler
	spool                *AttachmentSpool
	logger               *slog.Logger
	retention            time.Duration
	cleanupInterval      time.Duration
	cleanupBatchSize     int
	spoolTTL             time.Duration
	spoolCleanupInterval time.Duration
	spoolCleanupBatch    int
}

func (*Runtime) ModuleCode() kernel.ModuleCode { return ModuleCode }
func (r *Runtime) Mail() *Service              { return r.service }

func (r *Runtime) SiteManagementHTTP() httptransport.SiteManagementContribution {
	return httptransport.SiteManagementContribution{Path: "mail", Handler: r.managementHTTP}
}

func (r *Runtime) Jobs() []job.Definition {
	return []job.Definition{{Name: SendJobName, ScopeID: fmt.Sprint(r.service.siteID), Handler: r.worker.Handle}}
}

func (r *Runtime) BackgroundTasks() []background.Task {
	result := make([]background.Task, 0, 2)
	if r.retention > 0 {
		result = append(result, background.Task{Name: "mail.history_retention", Run: r.runRetention})
	}
	if r.spool != nil {
		result = append(result, background.Task{Name: "mail.spool_cleanup", Run: r.runSpoolCleanup})
	}
	return result
}

func (r *Runtime) runSpoolCleanup(ctx context.Context) error {
	cleanup := func() error {
		_, err := r.spool.Cleanup(ctx, time.Now().UTC().Add(-r.spoolTTL), r.spoolCleanupBatch, func(ctx context.Context, keys []string) (map[string]struct{}, error) {
			return r.service.repository.ActiveSpoolKeys(ctx, r.service.siteID, keys)
		})
		if err != nil && r.logger != nil {
			r.logger.ErrorContext(ctx, "mail spool cleanup failed", slog.String("event", "mail.spool.cleanup.failed"), slog.Any("error", err))
		}
		return err
	}
	if err := cleanup(); err != nil && ctx.Err() == nil {
		return err
	}
	ticker := time.NewTicker(r.spoolCleanupInterval)
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

func (r *Runtime) runRetention(ctx context.Context) error {
	cleanup := func() error {
		_, err := r.service.repository.Cleanup(ctx, r.service.siteID, r.retention, r.cleanupBatchSize)
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
	if config.SendMaxAttempts < 1 {
		return Config{}, errors.New("mail send maximum attempts is invalid")
	}
	if config.MaxRecipients < 1 || config.MaxMessageSize < 1 || config.MaxAttachmentSize < 1 {
		return Config{}, errors.New("mail delivery limits are invalid")
	}
	if strings.TrimSpace(string(config.UploadStorage)) == "" {
		config.UploadStorage = "private"
	}
	config.UploadPath = strings.Trim(strings.TrimSpace(config.UploadPath), "/")
	if config.UploadPath == "" {
		config.UploadPath = "mail"
	}
	if path.IsAbs(config.UploadPath) || path.Clean(config.UploadPath) != config.UploadPath || config.UploadPath == ".." || strings.HasPrefix(config.UploadPath, "../") || strings.Contains(config.UploadPath, "\\") {
		return Config{}, errors.New("mail upload path is invalid")
	}
	if config.SpoolEnabled {
		if config.SpoolTTL < time.Minute || config.SpoolCleanupInterval < time.Second || config.SpoolCleanupBatch < 1 {
			return Config{}, errors.New("mail spool configuration is invalid")
		}
	}
	return config, nil
}

var _ kernel.Module = Module{}
var _ kernel.DependencyProvider = Module{}
var _ kernel.ModuleDescriptorProvider = Module{}
var _ kernel.RegistryProvider = Module{}
var _ job.NamesProvider = Module{}
var _ kernel.ModuleRuntime = (*Runtime)(nil)
var _ job.Provider = (*Runtime)(nil)
var _ background.Provider = (*Runtime)(nil)
var _ adminui.NavigationProvider = (*Runtime)(nil)
var _ httptransport.SiteManagementProvider = (*Runtime)(nil)
