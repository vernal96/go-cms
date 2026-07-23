package core

import (
	"context"
	"errors"
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
)

const ModuleCode kernel.ModuleCode = "core"

const RepositoryCacheAlias cache.Alias = "repository"

const defaultRepositoryCacheTTL = 5 * time.Minute

type Config struct {
	RepositoryCacheTTL time.Duration
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

type Module struct{}

func (Module) Code() kernel.ModuleCode {
	return ModuleCode
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

func (Module) Build(
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
		store, exists := caches.Store(RepositoryCacheAlias)
		if exists {
			binding, bindingExists := caches.Binding(RepositoryCacheAlias)
			if !bindingExists {
				return nil, errors.New(
					"core repository cache binding is unavailable",
				)
			}
			database = newCachedDatabase(
				database,
				store,
				config.RepositoryCacheTTL,
			)
			descriptor = &RepositoryCacheDescriptor{
				Code:      binding.Code,
				Namespace: binding.Namespace,
				TTL:       config.RepositoryCacheTTL,
			}
		}
	}

	return &Runtime{
		database:        database,
		repositoryCache: descriptor,
	}, nil
}

type Runtime struct {
	database        Database
	repositoryCache *RepositoryCacheDescriptor
}

func (r *Runtime) ModuleCode() kernel.ModuleCode {
	return ModuleCode
}

func (r *Runtime) Database() Database {
	return r.database
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
var _ kernel.RegistryProvider = Module{}
var _ kernel.ModuleRuntime = (*Runtime)(nil)
