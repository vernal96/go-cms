package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/modules/core/access"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/group"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

const ModuleCode kernel.ModuleCode = "core"

const (
	DurableCacheAlias cache.Alias = "durable"
	HotCacheAlias     cache.Alias = "hot"
)

const defaultRepositoryCacheTTL = 5 * time.Minute

type Config struct {
	RepositoryCacheTTL time.Duration
	ResourcePreview    resource.PreviewPolicy
}

type RepositoryCacheDescriptor struct {
	Code      cache.Code
	Namespace string
	TTL       time.Duration
}

// Database is the persistence boundary required by the core module.
// Its concrete implementation is selected by the main application binding.
type Database interface {
	kernel.ModuleDatabase
	Sites() site.Repository
	Resources() resource.Repository
	Files() file.Repository
	Media() media.Repository
	Users() user.Repository
	Groups() group.Repository
	Access() access.Repository
}

type Module struct {
	services *Services
}

// BindServices returns the core module declaration bound to the
// application-scoped core services assembled by the composition root.
func BindServices(module kernel.Module, services *Services) (kernel.Module, error) {
	coreModule, ok := module.(Module)
	if !ok {
		return nil, fmt.Errorf("core module has invalid type %T", module)
	}
	if services == nil {
		return nil, errors.New("core services are nil")
	}
	coreModule.services = services
	return coreModule, nil
}

func (Module) Code() kernel.ModuleCode {
	return ModuleCode
}

func (Module) ModuleDescriptor() kernel.ModuleDescriptor {
	return kernel.ModuleDescriptor{
		Label:       "Core",
		Description: "Базовые возможности управления содержимым",
	}
}

func (Module) Registry() kernel.ModuleRegistry {
	return kernel.ModuleRegistry{
		FieldTypes:    field.StandardTypes(),
		ResourceTypes: resourcetype.StandardTypes(),
		PermissionEntities: []permission.Entity{
			{Code: "site"},
			{Code: "resource"},
			{Code: "file"},
			{Code: "media"},
			{Code: "user"},
			{Code: "group"},
		},
	}
}

func (m Module) Build(
	_ context.Context,
	ctx kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	database, err := kernel.ModuleDatabaseFrom[Database](
		ctx,
		"",
		ModuleCode,
	)
	if err != nil {
		return nil, err
	}

	if database.Sites() == nil {
		return nil, errors.New("core site repository is nil")
	}
	if database.Resources() == nil {
		return nil, errors.New("core resource repository is nil")
	}
	if database.Files() == nil {
		return nil, errors.New("core file repository is nil")
	}
	if database.Media() == nil {
		return nil, errors.New("core media repository is nil")
	}
	if database.Users() == nil {
		return nil, errors.New("core user repository is nil")
	}
	if database.Groups() == nil {
		return nil, errors.New("core group repository is nil")
	}
	if database.Access() == nil {
		return nil, errors.New("core access repository is nil")
	}
	if ModuleCode != ctx.ModuleCode() {
		return nil, errors.New("core module context has invalid code")
	}
	if m.services == nil {
		return nil, errors.New("core services are not bound")
	}

	config, err := kernel.ModuleConfigFrom[Config](ctx)
	if err != nil {
		return nil, err
	}
	if config.RepositoryCacheTTL == 0 {
		config.RepositoryCacheTTL = defaultRepositoryCacheTTL
	}
	if config.RepositoryCacheTTL < 0 {
		return nil, errors.New("core repository cache TTL is invalid")
	}

	var descriptor *RepositoryCacheDescriptor
	if caches := ctx.Caches(); caches != nil {
		store, exists := caches.Store(DurableCacheAlias)
		if exists {
			binding, bindingExists := caches.Binding(DurableCacheAlias)
			if !bindingExists {
				return nil, errors.New(
					"core repository cache binding is unavailable",
				)
			}
			descriptor = &RepositoryCacheDescriptor{
				Code:      binding.Code,
				Namespace: binding.Namespace,
				TTL:       config.RepositoryCacheTTL,
			}
			database = newCachedDatabase(
				database,
				store,
				config.RepositoryCacheTTL,
				m.services.cachePolicy,
			)
		}
	}

	return &Runtime{
		database:        database,
		repositoryCache: descriptor,
		services:        m.services,
		authorization:   m.services.Authorization,
		resourcePreview: config.ResourcePreview,
		logger:          ctx.Logger(),
	}, nil
}

type Runtime struct {
	database        Database
	repositoryCache *RepositoryCacheDescriptor
	services        *Services
	authorization   security.Authorizer
	resourcePreview resource.PreviewPolicy
	logger          *slog.Logger
}

func (r *Runtime) ModuleCode() kernel.ModuleCode {
	return ModuleCode
}

func (r *Runtime) Database() Database {
	return r.database
}

func (r *Runtime) Users() user.Service {
	if r == nil || r.services == nil {
		return nil
	}
	return r.services.Users
}

func (r *Runtime) Authorization() security.Authorizer {
	if r == nil || r.services == nil {
		return nil
	}
	return r.services.Authorization
}

func (r *Runtime) RepositoryCache() (
	RepositoryCacheDescriptor,
	bool,
) {
	if r == nil || r.repositoryCache == nil {
		return RepositoryCacheDescriptor{}, false
	}
	return *r.repositoryCache, true
}

var _ kernel.Module = Module{}
var _ kernel.ModuleDescriptorProvider = Module{}
var _ kernel.RegistryProvider = Module{}
var _ kernel.ModuleRuntime = (*Runtime)(nil)
