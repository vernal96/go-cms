package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/console"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/migrations"
	"github.com/vernal96/go-cms/kernel/modules/core"
	coreaccess "github.com/vernal96/go-cms/kernel/modules/core/access"
	corecommands "github.com/vernal96/go-cms/kernel/modules/core/commands"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	corefile "github.com/vernal96/go-cms/kernel/modules/core/file"
	coregroup "github.com/vernal96/go-cms/kernel/modules/core/group"
	coremedia "github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	coreuser "github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/modules/core/user/adapters/argon2id"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
	"github.com/vernal96/go-cms/kernel/seeds"
)

var (
	ErrClosed    = errors.New("app is closed")
	ErrNotBooted = errors.New("app is not booted")
)

type ConnectorFactory = kernel.ConnectorFactory
type ModuleDatabaseFactory = kernel.ModuleDatabaseFactory

type DatabaseDefinition struct {
	Connector ConnectorFactory
	Adapters  []ModuleDatabaseFactory
}

type Definition struct {
	MainDatabase        DatabaseDefinition
	AdditionalDatabases []DatabaseDefinition
	Filesystems         []filesystem.Factory
	Caches              []cache.Factory
	Profiles            []kernel.Profile
}

type bindingRuntime struct {
	connector kernel.DBConnector
	adapters  map[kernel.ModuleCode]kernel.ModuleDatabase
}

type App struct {
	definition Definition

	main          *bindingRuntime
	additional    map[kernel.ConnectionCode]*bindingRuntime
	connectors    []kernel.DBConnector
	filesystems   *filesystem.Manager
	caches        *cache.Manager
	coreDatabase  core.Database
	migrationPlan []migrations.Plan
	seedPlan      []seeds.Plan
	providers     []console.Provider
	console       *console.Console

	profileRuntimes map[kernel.ProfileCode]*kernel.ProfileRuntime
	sites           *site.Catalog
	resources       *resource.Service
	files           corefile.Service
	media           coremedia.Service
	users           coreuser.Service
	groups          coregroup.Service
	authorization   coreaccess.Service
	permissions     *permission.Catalog

	bootOnce sync.Once
	bootErr  error
	booted   atomic.Bool
	closed   atomic.Bool

	lifecycleMu sync.RWMutex
	closeOnce   sync.Once
	closeErr    error
}

func New(
	ctx context.Context,
	definition Definition,
) (_ *App, resultErr error) {
	if ctx == nil {
		return nil, errors.New("app context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	definition = cloneDefinition(definition)
	if err := validateDefinition(definition); err != nil {
		return nil, err
	}

	application := &App{
		definition: definition,
		additional: make(map[kernel.ConnectionCode]*bindingRuntime),
	}

	defer func() {
		if resultErr == nil {
			return
		}

		resultErr = errors.Join(resultErr, application.Close())
	}()

	filesystems, err := filesystem.NewManager(ctx, definition.Filesystems)
	if err != nil {
		return nil, err
	}
	application.filesystems = filesystems

	caches, err := cache.NewManager(
		ctx,
		definition.Caches,
		cache.Dependencies{Filesystems: filesystems},
	)
	if err != nil {
		return nil, err
	}
	application.caches = caches

	definitions := make(
		[]DatabaseDefinition,
		0,
		len(definition.AdditionalDatabases)+1,
	)
	definitions = append(definitions, definition.MainDatabase)
	definitions = append(definitions, definition.AdditionalDatabases...)

	for index, databaseDefinition := range definitions {
		binding, err := application.openBinding(ctx, databaseDefinition)
		if err != nil {
			return nil, err
		}

		if index == 0 {
			application.main = binding
		} else {
			application.additional[binding.connector.Code()] = binding
		}
	}

	coreAdapter, exists := application.main.adapters[core.ModuleCode]
	if !exists {
		return nil, errors.New(
			"main database binding does not contain core database",
		)
	}

	coreDatabase, ok := coreAdapter.(core.Database)
	if !ok {
		return nil, fmt.Errorf(
			"main database adapter %q does not implement core.Database",
			core.ModuleCode,
		)
	}
	if coreDatabase.Sites() == nil {
		return nil, errors.New("main core database has nil site repository")
	}
	if coreDatabase.Resources() == nil {
		return nil, errors.New(
			"main core database has nil resource repository",
		)
	}
	if coreDatabase.Files() == nil {
		return nil, errors.New(
			"main core database has nil file repository",
		)
	}
	if coreDatabase.Media() == nil {
		return nil, errors.New(
			"main core database has nil media repository",
		)
	}
	if coreDatabase.Users() == nil {
		return nil, errors.New(
			"main core database has nil user repository",
		)
	}
	if coreDatabase.Groups() == nil {
		return nil, errors.New(
			"main core database has nil group repository",
		)
	}
	if coreDatabase.Access() == nil {
		return nil, errors.New(
			"main core database has nil access repository",
		)
	}
	application.coreDatabase = coreDatabase

	permissionCatalog, err := buildPermissionCatalog(definition.Profiles)
	if err != nil {
		return nil, err
	}
	application.permissions = permissionCatalog

	application.collectModuleCommandProviders()
	application.addProvider(
		"core:identity-commands",
		corecommands.New(application),
	)

	runner, err := console.New(application)
	if err != nil {
		return nil, err
	}
	application.console = runner

	return application, nil
}

func (a *App) Boot(ctx context.Context) error {
	if a == nil {
		return errors.New("app is nil")
	}

	a.bootOnce.Do(func() {
		a.bootErr = a.boot(ctx)
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

	accessService, err := coreaccess.NewService(
		a.coreDatabase.Access(),
		a.permissions,
	)
	if err != nil {
		return err
	}

	fileService, err := corefile.NewService(
		a.coreDatabase.Files(),
		a.filesystems,
		accessService,
	)
	if err != nil {
		return err
	}

	mediaService, err := coremedia.NewService(
		a.coreDatabase.Media(),
		fileService,
		coremedia.FilePolicies{
			resource.ImageMediaUsage:  resource.ValidateImageMediaFile,
			coreuser.AvatarMediaUsage: coreuser.ValidateAvatarMediaFile,
		},
		accessService,
	)
	if err != nil {
		return err
	}

	groupService, err := coregroup.NewService(
		a.coreDatabase.Groups(),
		accessService,
	)
	if err != nil {
		return err
	}

	passwordHasher, err := argon2id.New()
	if err != nil {
		return err
	}
	userService, err := coreuser.NewService(
		a.coreDatabase.Users(),
		passwordHasher,
		mediaService,
		accessService,
	)
	if err != nil {
		return err
	}

	factory, err := kernel.NewProfileRuntimeFactory(
		a,
		kernel.RuntimeServices{
			Files:         fileService,
			Media:         mediaService,
			Users:         userService,
			Groups:        groupService,
			Authorization: accessService,
			Caches:        a.caches,
		},
	)
	if err != nil {
		return err
	}

	profileRuntimes := make(
		map[kernel.ProfileCode]*kernel.ProfileRuntime,
		len(a.definition.Profiles),
	)

	for _, profile := range a.definition.Profiles {
		runtime, err := factory.Make(ctx, profile)
		if err != nil {
			return fmt.Errorf(
				"compile profile runtime %q: %w",
				profile.Code,
				err,
			)
		}

		if _, exists := runtime.Registry().Module(core.ModuleCode); !exists {
			return fmt.Errorf(
				"profile %q does not contain required module %q",
				profile.Code,
				core.ModuleCode,
			)
		}

		profileRuntimes[profile.Code] = runtime
	}

	runtimeCoreDatabase, err := sharedCoreDatabase(
		a.definition.Profiles,
		profileRuntimes,
	)
	if err != nil {
		return err
	}

	catalog, err := site.NewCatalog(
		runtimeCoreDatabase.Sites(),
		profileResolver(profileRuntimes),
		accessService,
	)
	if err != nil {
		return err
	}

	if err := catalog.Reload(ctx); err != nil {
		return fmt.Errorf("compile site runtimes: %w", err)
	}

	resourceService, err := resource.NewService(
		runtimeCoreDatabase.Resources(),
		catalog,
		mediaService,
		accessService,
	)
	if err != nil {
		return err
	}

	a.profileRuntimes = profileRuntimes
	a.sites = catalog
	a.resources = resourceService
	a.files = fileService
	a.media = mediaService
	a.users = userService
	a.groups = groupService
	a.authorization = accessService
	a.booted.Store(true)
	return nil
}

func (a *App) Definition() Definition {
	if a == nil {
		return Definition{}
	}

	return cloneDefinition(a.definition)
}

func (a *App) Console() *console.Console {
	if a == nil {
		return nil
	}

	return a.console
}

func (a *App) ProfileRuntime(
	code kernel.ProfileCode,
) (*kernel.ProfileRuntime, bool) {
	if a == nil || a.closed.Load() || !a.booted.Load() {
		return nil, false
	}

	runtime, exists := a.profileRuntimes[code]
	return runtime, exists
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

func (a *App) UpdateSite(
	ctx context.Context,
	actor security.Actor,
	input site.UpdateInput,
) (*site.Runtime, error) {
	if a == nil {
		return nil, errors.New("app is nil")
	}
	if ctx == nil {
		return nil, errors.New("site settings update context is nil")
	}
	if a.closed.Load() {
		return nil, ErrClosed
	}
	if !a.booted.Load() {
		return nil, ErrNotBooted
	}

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()

	if a.closed.Load() {
		return nil, ErrClosed
	}

	return a.sites.Update(ctx, actor, input)
}

func (a *App) CreateResource(
	ctx context.Context,
	actor security.Actor,
	input resource.CreateInput,
) (resource.Resource, error) {
	service, err := a.resourceService()
	if err != nil {
		return resource.Resource{}, err
	}

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return resource.Resource{}, ErrClosed
	}

	return service.Create(ctx, actor, input)
}

func (a *App) Resource(
	ctx context.Context,
	actor security.Actor,
	id resource.ID,
) (resource.Resource, error) {
	service, err := a.resourceService()
	if err != nil {
		return resource.Resource{}, err
	}

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return resource.Resource{}, ErrClosed
	}

	return service.Get(ctx, actor, id)
}

func (a *App) ResourceByPath(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	path string,
) (resource.Resource, error) {
	service, err := a.resourceService()
	if err != nil {
		return resource.Resource{}, err
	}

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return resource.Resource{}, ErrClosed
	}

	return service.GetByPath(ctx, actor, siteID, path)
}

func (a *App) ResourceTree(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
) ([]resource.Node, error) {
	service, err := a.resourceService()
	if err != nil {
		return nil, err
	}

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return nil, ErrClosed
	}

	return service.Tree(ctx, actor, siteID)
}

func (a *App) UpdateResource(
	ctx context.Context,
	actor security.Actor,
	input resource.UpdateInput,
) (resource.Resource, error) {
	service, err := a.resourceService()
	if err != nil {
		return resource.Resource{}, err
	}

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return resource.Resource{}, ErrClosed
	}

	return service.Update(ctx, actor, input)
}

func (a *App) DeleteResource(
	ctx context.Context,
	actor security.Actor,
	id resource.ID,
) error {
	service, err := a.resourceService()
	if err != nil {
		return err
	}

	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return ErrClosed
	}

	return service.Delete(ctx, actor, id)
}

func (a *App) resourceService() (*resource.Service, error) {
	if a == nil {
		return nil, errors.New("app is nil")
	}
	if a.closed.Load() {
		return nil, ErrClosed
	}
	if !a.booted.Load() {
		return nil, ErrNotBooted
	}
	if a.resources == nil {
		return nil, errors.New("resource service is nil")
	}

	return a.resources, nil
}

func (a *App) CreateMedia(
	ctx context.Context,
	actor security.Actor,
	input coremedia.CreateInput,
) (coremedia.Media, error) {
	service, err := a.mediaService()
	if err != nil {
		return coremedia.Media{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coremedia.Media{}, ErrClosed
	}
	return service.Create(ctx, actor, input)
}

func (a *App) Media(
	ctx context.Context,
	actor security.Actor,
	id coremedia.ID,
) (coremedia.Media, error) {
	service, err := a.mediaService()
	if err != nil {
		return coremedia.Media{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coremedia.Media{}, ErrClosed
	}
	return service.Get(ctx, actor, id)
}

func (a *App) UpdateMedia(
	ctx context.Context,
	actor security.Actor,
	input coremedia.UpdateInput,
) (coremedia.Media, error) {
	service, err := a.mediaService()
	if err != nil {
		return coremedia.Media{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coremedia.Media{}, ErrClosed
	}
	return service.Update(ctx, actor, input)
}

func (a *App) DeleteMedia(
	ctx context.Context,
	actor security.Actor,
	id coremedia.ID,
) error {
	service, err := a.mediaService()
	if err != nil {
		return err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return ErrClosed
	}
	return service.Delete(ctx, actor, id)
}

func (a *App) mediaService() (coremedia.Service, error) {
	if a == nil {
		return nil, errors.New("app is nil")
	}
	if a.closed.Load() {
		return nil, ErrClosed
	}
	if !a.booted.Load() {
		return nil, ErrNotBooted
	}
	if a.media == nil {
		return nil, errors.New("media service is nil")
	}
	return a.media, nil
}

func (a *App) CreateFileFolder(
	ctx context.Context,
	actor security.Actor,
	input corefile.CreateFolderInput,
) (corefile.Folder, error) {
	service, err := a.fileService()
	if err != nil {
		return corefile.Folder{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return corefile.Folder{}, ErrClosed
	}
	return service.CreateFolder(ctx, actor, input)
}

func (a *App) FileFolder(
	ctx context.Context,
	actor security.Actor,
	id corefile.FolderID,
) (corefile.Folder, error) {
	service, err := a.fileService()
	if err != nil {
		return corefile.Folder{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return corefile.Folder{}, ErrClosed
	}
	return service.GetFolder(ctx, actor, id)
}

func (a *App) ListFileFolder(
	ctx context.Context,
	actor security.Actor,
	storage filesystem.Code,
	id *corefile.FolderID,
) (corefile.Listing, error) {
	service, err := a.fileService()
	if err != nil {
		return corefile.Listing{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return corefile.Listing{}, ErrClosed
	}
	return service.ListFolder(ctx, actor, storage, id)
}

func (a *App) UploadFile(
	ctx context.Context,
	actor security.Actor,
	input corefile.UploadInput,
) (corefile.File, error) {
	service, err := a.fileService()
	if err != nil {
		return corefile.File{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return corefile.File{}, ErrClosed
	}
	return service.Upload(ctx, actor, input)
}

func (a *App) File(
	ctx context.Context,
	actor security.Actor,
	id corefile.ID,
) (corefile.File, error) {
	service, err := a.fileService()
	if err != nil {
		return corefile.File{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return corefile.File{}, ErrClosed
	}
	return service.GetFile(ctx, actor, id)
}

func (a *App) OpenFile(
	ctx context.Context,
	actor security.Actor,
	id corefile.ID,
) (corefile.OpenedFile, error) {
	service, err := a.fileService()
	if err != nil {
		return corefile.OpenedFile{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return corefile.OpenedFile{}, ErrClosed
	}
	return service.Open(ctx, actor, id)
}

func (a *App) OpenFileDelivery(
	ctx context.Context,
	id corefile.ID,
	authorization corefile.DeliveryAuthorization,
) (corefile.OpenedFile, error) {
	service, err := a.fileService()
	if err != nil {
		return corefile.OpenedFile{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return corefile.OpenedFile{}, ErrClosed
	}
	return service.OpenDelivery(ctx, id, authorization)
}

func (a *App) MoveFile(
	ctx context.Context,
	actor security.Actor,
	input corefile.MoveFileInput,
) (corefile.File, error) {
	service, err := a.fileService()
	if err != nil {
		return corefile.File{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return corefile.File{}, ErrClosed
	}
	return service.MoveFile(ctx, actor, input)
}

func (a *App) MoveFileFolder(
	ctx context.Context,
	actor security.Actor,
	input corefile.MoveFolderInput,
) (corefile.Folder, error) {
	service, err := a.fileService()
	if err != nil {
		return corefile.Folder{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return corefile.Folder{}, ErrClosed
	}
	return service.MoveFolder(ctx, actor, input)
}

func (a *App) DeleteFile(
	ctx context.Context,
	actor security.Actor,
	id corefile.ID,
) error {
	service, err := a.fileService()
	if err != nil {
		return err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return ErrClosed
	}
	return service.DeleteFile(ctx, actor, id)
}

func (a *App) DeleteFileFolder(
	ctx context.Context,
	actor security.Actor,
	id corefile.FolderID,
) error {
	service, err := a.fileService()
	if err != nil {
		return err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return ErrClosed
	}
	return service.DeleteFolder(ctx, actor, id)
}

func (a *App) FileURL(
	ctx context.Context,
	actor security.Actor,
	id corefile.ID,
) (string, error) {
	service, err := a.fileService()
	if err != nil {
		return "", err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return "", ErrClosed
	}
	return service.URL(ctx, actor, id)
}

func (a *App) TemporaryFileURL(
	ctx context.Context,
	actor security.Actor,
	id corefile.ID,
	expiresAt time.Time,
) (string, error) {
	service, err := a.fileService()
	if err != nil {
		return "", err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return "", ErrClosed
	}
	return service.TemporaryURL(ctx, actor, id, expiresAt)
}

func (a *App) fileService() (corefile.Service, error) {
	if a == nil {
		return nil, errors.New("app is nil")
	}
	if a.closed.Load() {
		return nil, ErrClosed
	}
	if !a.booted.Load() {
		return nil, ErrNotBooted
	}
	if a.files == nil {
		return nil, errors.New("file service is nil")
	}
	return a.files, nil
}

func (a *App) CreateUser(
	ctx context.Context,
	actor security.Actor,
	input coreuser.CreateInput,
) (coreuser.User, error) {
	service, err := a.userService()
	if err != nil {
		return coreuser.User{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coreuser.User{}, ErrClosed
	}
	return service.Create(ctx, actor, input)
}

func (a *App) User(
	ctx context.Context,
	actor security.Actor,
	id coreuser.ID,
) (coreuser.User, error) {
	service, err := a.userService()
	if err != nil {
		return coreuser.User{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coreuser.User{}, ErrClosed
	}
	return service.Get(ctx, actor, id)
}

func (a *App) Users(
	ctx context.Context,
	actor security.Actor,
) ([]coreuser.User, error) {
	service, err := a.userService()
	if err != nil {
		return nil, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return nil, ErrClosed
	}
	return service.List(ctx, actor)
}

func (a *App) UpdateUser(
	ctx context.Context,
	actor security.Actor,
	input coreuser.UpdateInput,
) (coreuser.User, error) {
	service, err := a.userService()
	if err != nil {
		return coreuser.User{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coreuser.User{}, ErrClosed
	}
	return service.Update(ctx, actor, input)
}

func (a *App) ChangeUserPassword(
	ctx context.Context,
	actor security.Actor,
	id coreuser.ID,
	password string,
) (coreuser.User, error) {
	service, err := a.userService()
	if err != nil {
		return coreuser.User{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coreuser.User{}, ErrClosed
	}
	return service.ChangePassword(ctx, actor, id, password)
}

func (a *App) DeleteUser(
	ctx context.Context,
	actor security.Actor,
	id coreuser.ID,
) (coreuser.User, error) {
	service, err := a.userService()
	if err != nil {
		return coreuser.User{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coreuser.User{}, ErrClosed
	}
	return service.Delete(ctx, actor, id)
}

func (a *App) RestoreUser(
	ctx context.Context,
	actor security.Actor,
	id coreuser.ID,
) (coreuser.User, error) {
	service, err := a.userService()
	if err != nil {
		return coreuser.User{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coreuser.User{}, ErrClosed
	}
	return service.Restore(ctx, actor, id)
}

func (a *App) Authenticate(
	ctx context.Context,
	input coreuser.AuthenticateInput,
) (coreuser.User, error) {
	service, err := a.userService()
	if err != nil {
		return coreuser.User{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coreuser.User{}, ErrClosed
	}
	return service.Authenticate(ctx, input)
}

func (a *App) CreateGroup(
	ctx context.Context,
	actor security.Actor,
	input coregroup.CreateInput,
) (coregroup.Group, error) {
	service, err := a.groupService()
	if err != nil {
		return coregroup.Group{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coregroup.Group{}, ErrClosed
	}
	return service.Create(ctx, actor, input)
}

func (a *App) Group(
	ctx context.Context,
	actor security.Actor,
	id coregroup.ID,
) (coregroup.Group, error) {
	service, err := a.groupService()
	if err != nil {
		return coregroup.Group{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coregroup.Group{}, ErrClosed
	}
	return service.Get(ctx, actor, id)
}

func (a *App) Groups(
	ctx context.Context,
	actor security.Actor,
) ([]coregroup.Group, error) {
	service, err := a.groupService()
	if err != nil {
		return nil, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return nil, ErrClosed
	}
	return service.List(ctx, actor)
}

func (a *App) UpdateGroup(
	ctx context.Context,
	actor security.Actor,
	input coregroup.UpdateInput,
) (coregroup.Group, error) {
	service, err := a.groupService()
	if err != nil {
		return coregroup.Group{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coregroup.Group{}, ErrClosed
	}
	return service.Update(ctx, actor, input)
}

func (a *App) DeleteGroup(
	ctx context.Context,
	actor security.Actor,
	id coregroup.ID,
) error {
	service, err := a.groupService()
	if err != nil {
		return err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return ErrClosed
	}
	return service.Delete(ctx, actor, id)
}

func (a *App) AddUserToGroup(
	ctx context.Context,
	actor security.Actor,
	groupID coregroup.ID,
	userID security.UserID,
) (coregroup.Membership, error) {
	service, err := a.groupService()
	if err != nil {
		return coregroup.Membership{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coregroup.Membership{}, ErrClosed
	}
	return service.AddUser(ctx, actor, groupID, userID)
}

func (a *App) RemoveUserFromGroup(
	ctx context.Context,
	actor security.Actor,
	groupID coregroup.ID,
	userID security.UserID,
) error {
	service, err := a.groupService()
	if err != nil {
		return err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return ErrClosed
	}
	return service.RemoveUser(ctx, actor, groupID, userID)
}

func (a *App) GroupMembers(
	ctx context.Context,
	actor security.Actor,
	groupID coregroup.ID,
) ([]coregroup.Membership, error) {
	service, err := a.groupService()
	if err != nil {
		return nil, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return nil, ErrClosed
	}
	return service.Members(ctx, actor, groupID)
}

func (a *App) GroupPermissions(
	ctx context.Context,
	actor security.Actor,
	groupID coregroup.ID,
) ([]coregroup.PermissionGrant, error) {
	service, err := a.groupService()
	if err != nil {
		return nil, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return nil, ErrClosed
	}
	return service.Permissions(ctx, actor, groupID)
}

func (a *App) GrantGroupPermission(
	ctx context.Context,
	actor security.Actor,
	groupID coregroup.ID,
	code permission.Code,
) (coregroup.PermissionGrant, error) {
	service, err := a.groupService()
	if err != nil {
		return coregroup.PermissionGrant{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coregroup.PermissionGrant{}, ErrClosed
	}
	return service.GrantPermission(ctx, actor, groupID, code)
}

func (a *App) RevokeGroupPermission(
	ctx context.Context,
	actor security.Actor,
	groupID coregroup.ID,
	code permission.Code,
) error {
	service, err := a.groupService()
	if err != nil {
		return err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return ErrClosed
	}
	return service.RevokePermission(ctx, actor, groupID, code)
}

func (a *App) PermissionCodes() ([]permission.Code, error) {
	if _, err := a.accessService(); err != nil {
		return nil, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return nil, ErrClosed
	}
	return a.permissions.Codes(), nil
}

func (a *App) GrantGuestPermission(
	ctx context.Context,
	actor security.Actor,
	code permission.Code,
) (coreaccess.Grant, error) {
	service, err := a.accessService()
	if err != nil {
		return coreaccess.Grant{}, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return coreaccess.Grant{}, ErrClosed
	}
	return service.GrantGuest(ctx, actor, code)
}

func (a *App) RevokeGuestPermission(
	ctx context.Context,
	actor security.Actor,
	code permission.Code,
) error {
	service, err := a.accessService()
	if err != nil {
		return err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return ErrClosed
	}
	return service.RevokeGuest(ctx, actor, code)
}

func (a *App) GuestPermissions(
	ctx context.Context,
	actor security.Actor,
) ([]coreaccess.Grant, error) {
	service, err := a.accessService()
	if err != nil {
		return nil, err
	}
	a.lifecycleMu.RLock()
	defer a.lifecycleMu.RUnlock()
	if a.closed.Load() {
		return nil, ErrClosed
	}
	return service.GuestPermissions(ctx, actor)
}

func (a *App) userService() (coreuser.Service, error) {
	if a == nil {
		return nil, errors.New("app is nil")
	}
	if a.closed.Load() {
		return nil, ErrClosed
	}
	if !a.booted.Load() {
		return nil, ErrNotBooted
	}
	if a.users == nil {
		return nil, errors.New("user service is nil")
	}
	return a.users, nil
}

func (a *App) groupService() (coregroup.Service, error) {
	if a == nil {
		return nil, errors.New("app is nil")
	}
	if a.closed.Load() {
		return nil, ErrClosed
	}
	if !a.booted.Load() {
		return nil, ErrNotBooted
	}
	if a.groups == nil {
		return nil, errors.New("group service is nil")
	}
	return a.groups, nil
}

func (a *App) accessService() (coreaccess.Service, error) {
	if a == nil {
		return nil, errors.New("app is nil")
	}
	if a.closed.Load() {
		return nil, ErrClosed
	}
	if !a.booted.Load() {
		return nil, ErrNotBooted
	}
	if a.authorization == nil {
		return nil, errors.New("access service is nil")
	}
	return a.authorization, nil
}

func (a *App) MigrationPlans() []migrations.Plan {
	if a == nil {
		return nil
	}

	return append([]migrations.Plan(nil), a.migrationPlan...)
}

func (a *App) SeedPlans() []seeds.Plan {
	if a == nil {
		return nil
	}

	plans := append([]seeds.Plan(nil), a.seedPlan...)
	for index := range plans {
		plans[index].Source.Tags = append(
			[]seeds.Tag(nil),
			plans[index].Source.Tags...,
		)
	}

	return plans
}

func (a *App) CommandProviders() []console.Provider {
	if a == nil {
		return nil
	}

	return append([]console.Provider(nil), a.providers...)
}

func (a *App) MainModuleDatabase(
	moduleCode kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	if a == nil || a.main == nil || a.closed.Load() {
		return nil, false
	}

	database, exists := a.main.adapters[moduleCode]
	return database, exists
}

func (a *App) ModuleDatabase(
	connectionCode kernel.ConnectionCode,
	moduleCode kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	if a == nil || a.closed.Load() {
		return nil, false
	}

	if connectionCode == "" || connectionCode == a.main.connector.Code() {
		return a.MainModuleDatabase(moduleCode)
	}

	binding, exists := a.additional[connectionCode]
	if !exists {
		return nil, false
	}

	database, exists := binding.adapters[moduleCode]
	return database, exists
}

func (a *App) Close() error {
	if a == nil {
		return nil
	}

	a.closeOnce.Do(func() {
		a.lifecycleMu.Lock()
		defer a.lifecycleMu.Unlock()

		a.closed.Store(true)

		var closeErrors []error
		for index := len(a.connectors) - 1; index >= 0; index-- {
			connector := a.connectors[index]
			if err := connector.Close(); err != nil {
				closeErrors = append(closeErrors, fmt.Errorf(
					"close database connector %q: %w",
					connector.Code(),
					err,
				))
			}
		}
		if a.caches != nil {
			if err := a.caches.Close(); err != nil {
				closeErrors = append(closeErrors, err)
			}
		}
		if a.filesystems != nil {
			if err := a.filesystems.Close(); err != nil {
				closeErrors = append(closeErrors, err)
			}
		}

		a.closeErr = errors.Join(closeErrors...)
	})

	return a.closeErr
}

func (a *App) openBinding(
	ctx context.Context,
	definition DatabaseDefinition,
) (*bindingRuntime, error) {
	connector, err := definition.Connector.Open(ctx)
	if connector != nil {
		a.connectors = append(a.connectors, connector)
	}
	if err != nil {
		return nil, fmt.Errorf(
			"open database connector %q: %w",
			definition.Connector.Code(),
			err,
		)
	}
	if connector == nil {
		return nil, fmt.Errorf(
			"database connector factory %q returned nil connector",
			definition.Connector.Code(),
		)
	}
	if connector.Code() != definition.Connector.Code() {
		return nil, fmt.Errorf(
			"database connector factory %q returned connector %q",
			definition.Connector.Code(),
			connector.Code(),
		)
	}
	if err := connector.Ping(ctx); err != nil {
		return nil, fmt.Errorf(
			"ping database connector %q: %w",
			connector.Code(),
			err,
		)
	}

	a.addProvider("connector:"+string(connector.Code()), connector)

	binding := &bindingRuntime{
		connector: connector,
		adapters:  make(map[kernel.ModuleCode]kernel.ModuleDatabase),
	}
	migrationSourceIDs := make(map[string]struct{})
	seedSourceIDs := make(map[string]struct{})
	seedHistories := make(map[string]string)

	for _, factory := range definition.Adapters {
		database, err := factory.Build(connector)
		if err != nil {
			return nil, fmt.Errorf(
				"build database adapter %q on connection %q: %w",
				factory.ModuleCode(),
				connector.Code(),
				err,
			)
		}
		if database == nil {
			return nil, fmt.Errorf(
				"database adapter factory %q returned nil",
				factory.ModuleCode(),
			)
		}
		if database.ModuleCode() != factory.ModuleCode() {
			return nil, fmt.Errorf(
				"database adapter factory %q returned adapter %q",
				factory.ModuleCode(),
				database.ModuleCode(),
			)
		}

		moduleCode := database.ModuleCode()
		binding.adapters[moduleCode] = database
		a.addProvider("database:"+string(moduleCode), database)

		if provider, ok := database.(migrations.Provider); ok {
			plans, err := migrationPlans(
				connector,
				moduleCode,
				provider.MigrationSources(),
				migrationSourceIDs,
			)
			if err != nil {
				return nil, err
			}
			a.migrationPlan = append(a.migrationPlan, plans...)
		}

		if provider, ok := database.(seeds.Provider); ok {
			plans, err := seedPlans(
				connector,
				moduleCode,
				provider.SeedSources(),
				seedSourceIDs,
				seedHistories,
			)
			if err != nil {
				return nil, err
			}
			a.seedPlan = append(a.seedPlan, plans...)
		}
	}

	return binding, nil
}

func (a *App) collectModuleCommandProviders() {
	for _, profile := range a.definition.Profiles {
		for _, profileModule := range profile.Modules {
			if profileModule.Module == nil {
				continue
			}

			a.addProvider(
				"module:"+string(profileModule.Module.Code()),
				profileModule.Module,
			)
		}
	}
}

func buildPermissionCatalog(
	profiles []kernel.Profile,
) (*permission.Catalog, error) {
	seenModules := make(map[kernel.ModuleCode]struct{})
	definitions := make([]permission.Definition, 0)

	for _, profile := range profiles {
		for _, profileModule := range profile.Modules {
			if profileModule.Module == nil {
				continue
			}
			moduleCode := profileModule.Module.Code()
			if _, exists := seenModules[moduleCode]; exists {
				continue
			}
			seenModules[moduleCode] = struct{}{}

			provider, exists := profileModule.Module.(kernel.RegistryProvider)
			if !exists {
				continue
			}
			moduleDefinitions, err := permission.Definitions(
				string(moduleCode),
				provider.Registry().PermissionEntities,
			)
			if err != nil {
				return nil, fmt.Errorf(
					"build permissions for module %q: %w",
					moduleCode,
					err,
				)
			}
			definitions = append(definitions, moduleDefinitions...)
		}
	}

	return permission.NewCatalog(definitions)
}

func (a *App) addProvider(key string, candidate any) {
	provider, ok := candidate.(console.Provider)
	if !ok {
		return
	}

	for _, registered := range a.providers {
		if providerIdentity(registered) == key {
			return
		}
	}

	a.providers = append(a.providers, keyedProvider{
		key:      key,
		Provider: provider,
	})
}

type keyedProvider struct {
	key string
	console.Provider
}

func providerIdentity(provider console.Provider) string {
	if keyed, ok := provider.(keyedProvider); ok {
		return keyed.key
	}
	return ""
}

type profileResolver map[kernel.ProfileCode]*kernel.ProfileRuntime

func (r profileResolver) ProfileRuntime(
	code kernel.ProfileCode,
) (*kernel.ProfileRuntime, bool) {
	runtime, exists := r[code]
	return runtime, exists
}

func sharedCoreDatabase(
	profiles []kernel.Profile,
	runtimes map[kernel.ProfileCode]*kernel.ProfileRuntime,
) (core.Database, error) {
	var (
		selected    core.Database
		expected    core.RepositoryCacheDescriptor
		hasExpected bool
		initialized bool
	)

	for _, profile := range profiles {
		runtime := runtimes[profile.Code]
		if runtime == nil {
			return nil, fmt.Errorf(
				"profile runtime %q is nil",
				profile.Code,
			)
		}
		moduleRuntime, exists := runtime.Registry().Module(core.ModuleCode)
		if !exists {
			return nil, fmt.Errorf(
				"profile %q does not contain required module %q",
				profile.Code,
				core.ModuleCode,
			)
		}
		coreRuntime, ok := moduleRuntime.(*core.Runtime)
		if !ok {
			return nil, fmt.Errorf(
				"profile %q core runtime has type %T",
				profile.Code,
				moduleRuntime,
			)
		}
		current, hasCurrent := coreRuntime.RepositoryCache()
		if !initialized {
			selected = coreRuntime.Database()
			expected = current
			hasExpected = hasCurrent
			initialized = true
			continue
		}
		if hasExpected != hasCurrent ||
			(hasExpected && expected != current) {
			return nil, errors.New(
				"all profiles must use the same core repository cache store, namespace, and TTL",
			)
		}
	}
	if !initialized || selected == nil {
		return nil, errors.New("core runtime database is unavailable")
	}
	return selected, nil
}

func migrationPlans(
	connector kernel.DBConnector,
	moduleCode kernel.ModuleCode,
	sources []migrations.Source,
	used map[string]struct{},
) ([]migrations.Plan, error) {
	if len(sources) == 0 {
		return nil, nil
	}

	target, ok := connector.(migrations.Target)
	if !ok {
		return nil, fmt.Errorf(
			"database connector %q does not support migrations required by module %q",
			connector.Code(),
			moduleCode,
		)
	}

	plans := make([]migrations.Plan, 0, len(sources))
	for _, source := range sources {
		if err := validateSource(connector.Code(), moduleCode, source.ID, used); err != nil {
			return nil, err
		}

		plans = append(plans, migrations.Plan{
			Connection: string(connector.Code()),
			Target:     target,
			Source:     source,
		})
	}

	return plans, nil
}

func seedPlans(
	connector kernel.DBConnector,
	moduleCode kernel.ModuleCode,
	sources []seeds.Source,
	used map[string]struct{},
	histories map[string]string,
) ([]seeds.Plan, error) {
	if len(sources) == 0 {
		return nil, nil
	}

	target, ok := connector.(seeds.Target)
	if !ok {
		return nil, fmt.Errorf(
			"database connector %q does not support seeds required by module %q",
			connector.Code(),
			moduleCode,
		)
	}

	plans := make([]seeds.Plan, 0, len(sources))
	for _, source := range sources {
		if err := validateSeedSource(
			connector.Code(),
			moduleCode,
			source,
			used,
			histories,
		); err != nil {
			return nil, err
		}

		source.Tags = append([]seeds.Tag(nil), source.Tags...)
		plans = append(plans, seeds.Plan{
			Connection: string(connector.Code()),
			Module:     moduleCode,
			Target:     target,
			Source:     source,
		})
	}

	return plans, nil
}

func validateSeedSource(
	connectionCode kernel.ConnectionCode,
	moduleCode kernel.ModuleCode,
	source seeds.Source,
	used map[string]struct{},
	histories map[string]string,
) error {
	if err := seeds.ValidateSource(source); err != nil {
		return fmt.Errorf(
			"database binding %q module %q: %w",
			connectionCode,
			moduleCode,
			err,
		)
	}

	sourceKey := string(moduleCode) + "/" + source.ID
	if _, exists := used[sourceKey]; exists {
		return fmt.Errorf(
			"database binding %q module %q contains duplicate seed source %q",
			connectionCode,
			moduleCode,
			source.ID,
		)
	}

	historyKey := source.Schema + "/" + seeds.HistoryTable(source.ID)
	if existing, exists := histories[historyKey]; exists {
		return fmt.Errorf(
			"database binding %q seed sources %q and %q share history %q",
			connectionCode,
			existing,
			sourceKey,
			historyKey,
		)
	}

	used[sourceKey] = struct{}{}
	histories[historyKey] = sourceKey
	return nil
}

func validateSource(
	connectionCode kernel.ConnectionCode,
	moduleCode kernel.ModuleCode,
	sourceID string,
	used map[string]struct{},
) error {
	if sourceID != string(moduleCode) {
		return fmt.Errorf(
			"database binding %q module %q returned source %q",
			connectionCode,
			moduleCode,
			sourceID,
		)
	}
	if _, exists := used[sourceID]; exists {
		return fmt.Errorf(
			"database binding %q contains duplicate source %q",
			connectionCode,
			sourceID,
		)
	}

	used[sourceID] = struct{}{}
	return nil
}

func validateDefinition(definition Definition) error {
	filesystemCodes := make(
		map[filesystem.Code]struct{},
		len(definition.Filesystems),
	)
	for index, factory := range definition.Filesystems {
		if factory == nil {
			return fmt.Errorf(
				"filesystem factory at index %d is nil",
				index,
			)
		}
		code := factory.Code()
		if code == "" {
			return fmt.Errorf(
				"filesystem factory at index %d has empty code",
				index,
			)
		}
		if _, exists := filesystemCodes[code]; exists {
			return fmt.Errorf(
				"filesystem disk %q is defined more than once",
				code,
			)
		}
		filesystemCodes[code] = struct{}{}
	}

	cacheCodes := make(
		map[cache.Code]struct{},
		len(definition.Caches),
	)
	for index, factory := range definition.Caches {
		if factory == nil {
			return fmt.Errorf(
				"cache factory at index %d is nil",
				index,
			)
		}
		code := factory.Code()
		if code == "" {
			return fmt.Errorf(
				"cache factory at index %d has empty code",
				index,
			)
		}
		if _, exists := cacheCodes[code]; exists {
			return fmt.Errorf(
				"cache store %q is defined more than once",
				code,
			)
		}
		cacheCodes[code] = struct{}{}
	}

	definitions := make(
		[]DatabaseDefinition,
		0,
		len(definition.AdditionalDatabases)+1,
	)
	definitions = append(definitions, definition.MainDatabase)
	definitions = append(definitions, definition.AdditionalDatabases...)

	connectionCodes := make(map[kernel.ConnectionCode]struct{}, len(definitions))
	for bindingIndex, database := range definitions {
		if database.Connector == nil {
			return fmt.Errorf(
				"database definition at index %d has nil connector factory",
				bindingIndex,
			)
		}

		connectionCode := database.Connector.Code()
		if connectionCode == "" {
			return fmt.Errorf(
				"database definition at index %d has empty connection code",
				bindingIndex,
			)
		}
		if _, exists := connectionCodes[connectionCode]; exists {
			return fmt.Errorf(
				"database connection %q is defined more than once",
				connectionCode,
			)
		}
		connectionCodes[connectionCode] = struct{}{}

		moduleCodes := make(map[kernel.ModuleCode]struct{}, len(database.Adapters))
		for adapterIndex, adapter := range database.Adapters {
			if adapter == nil {
				return fmt.Errorf(
					"database definition %q adapter at index %d is nil",
					connectionCode,
					adapterIndex,
				)
			}

			moduleCode := adapter.ModuleCode()
			if moduleCode == "" {
				return fmt.Errorf(
					"database definition %q adapter at index %d has empty module code",
					connectionCode,
					adapterIndex,
				)
			}
			if _, exists := moduleCodes[moduleCode]; exists {
				return fmt.Errorf(
					"database definition %q contains duplicate module %q",
					connectionCode,
					moduleCode,
				)
			}
			moduleCodes[moduleCode] = struct{}{}
		}
	}

	profileCodes := make(map[kernel.ProfileCode]struct{}, len(definition.Profiles))
	for profileIndex, profile := range definition.Profiles {
		if profile.Code == "" {
			return fmt.Errorf("profile at index %d has empty code", profileIndex)
		}
		if _, exists := profileCodes[profile.Code]; exists {
			return fmt.Errorf("profile %q is defined more than once", profile.Code)
		}
		profileCodes[profile.Code] = struct{}{}

		moduleCodes := make(map[kernel.ModuleCode]struct{}, len(profile.Modules))
		for moduleIndex, profileModule := range profile.Modules {
			if profileModule.Module == nil {
				return fmt.Errorf(
					"profile %q module at index %d is nil",
					profile.Code,
					moduleIndex,
				)
			}

			moduleCode := profileModule.Module.Code()
			if moduleCode == "" {
				return fmt.Errorf(
					"profile %q module at index %d has empty code",
					profile.Code,
					moduleIndex,
				)
			}
			if _, exists := moduleCodes[moduleCode]; exists {
				return fmt.Errorf(
					"profile %q contains duplicate module %q",
					profile.Code,
					moduleCode,
				)
			}
			moduleCodes[moduleCode] = struct{}{}
		}
	}

	return nil
}

func cloneDefinition(definition Definition) Definition {
	definition.Filesystems = append(
		[]filesystem.Factory(nil),
		definition.Filesystems...,
	)
	definition.Caches = append(
		[]cache.Factory(nil),
		definition.Caches...,
	)
	definition.MainDatabase.Adapters = append(
		[]ModuleDatabaseFactory(nil),
		definition.MainDatabase.Adapters...,
	)
	definition.AdditionalDatabases = append(
		[]DatabaseDefinition(nil),
		definition.AdditionalDatabases...,
	)
	for index := range definition.AdditionalDatabases {
		definition.AdditionalDatabases[index].Adapters = append(
			[]ModuleDatabaseFactory(nil),
			definition.AdditionalDatabases[index].Adapters...,
		)
	}

	definition.Profiles = append([]kernel.Profile(nil), definition.Profiles...)
	for index := range definition.Profiles {
		definition.Profiles[index].Modules = append(
			[]kernel.ProfileModule(nil),
			definition.Profiles[index].Modules...,
		)
		for moduleIndex := range definition.Profiles[index].Modules {
			definition.Profiles[index].Modules[moduleIndex].Caches = append(
				[]cache.Binding(nil),
				definition.Profiles[index].Modules[moduleIndex].Caches...,
			)
		}
		definition.Profiles[index].Params = field.CloneDefinitions(
			definition.Profiles[index].Params,
		)
		definition.Profiles[index].Templates = template.CloneDefinitions(
			definition.Profiles[index].Templates,
		)
	}

	return definition
}

var _ kernel.DatabaseResolver = (*App)(nil)
var _ console.Application = (*App)(nil)
