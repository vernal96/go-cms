package httpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vernal96/go-cms/connectors/localstorage"
	httpserver "github.com/vernal96/go-cms/internal/server/http"
	"github.com/vernal96/go-cms/kernel"
	appkernel "github.com/vernal96/go-cms/kernel/app"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/core"
	coreaccess "github.com/vernal96/go-cms/kernel/modules/core/access"
	corefile "github.com/vernal96/go-cms/kernel/modules/core/file"
	coregroup "github.com/vernal96/go-cms/kernel/modules/core/group"
	coremedia "github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	coreuser "github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
	httptransport "github.com/vernal96/go-cms/kernel/transport/http"
)

type connector struct{}

func (connector) Code() kernel.ConnectionCode { return "test" }
func (connector) Ping(context.Context) error  { return nil }
func (connector) Close() error                { return nil }

type connectorFactory struct{}

func (connectorFactory) Code() kernel.ConnectionCode { return "test" }
func (connectorFactory) Open(context.Context) (kernel.DBConnector, error) {
	return connector{}, nil
}

type repository struct {
	isPublic bool
}

func (r repository) List(context.Context) ([]site.Site, error) {
	return []site.Site{
		{
			ID:          1,
			ProfileCode: "dev",
			Domain:      "example.com",
			Locale:      "ru-RU",
			IsPublic:    r.isPublic,
		},
	}, nil
}

func (repository) Update(
	context.Context,
	*security.UserID,
	site.Site,
) (site.Site, error) {
	return site.Site{}, nil
}

type database struct {
	files     corefile.Repository
	sites     site.Repository
	resources resource.Repository
	access    coreaccess.Repository
}

func (database) ModuleCode() kernel.ModuleCode { return core.ModuleCode }
func (d database) Sites() site.Repository {
	if d.sites != nil {
		return d.sites
	}
	return repository{isPublic: true}
}
func (d database) Resources() resource.Repository {
	if d.resources != nil {
		return d.resources
	}
	return resourceRepository{}
}
func (d database) Files() corefile.Repository {
	if d.files != nil {
		return d.files
	}
	return fileRepository{}
}
func (database) Media() coremedia.Repository {
	return mediaRepository{}
}
func (database) Users() coreuser.Repository   { return userRepository{} }
func (database) Groups() coregroup.Repository { return groupRepository{} }
func (d database) Access() coreaccess.Repository {
	if d.access != nil {
		return d.access
	}
	return accessRepository{}
}

type mediaRepository struct {
	coremedia.Repository
}

type userRepository struct {
	coreuser.Repository
}

type groupRepository struct {
	coregroup.Repository
}

type accessRepository struct{}

func (accessRepository) Subject(
	context.Context,
	security.UserID,
) (coreaccess.Subject, error) {
	return coreaccess.Subject{}, nil
}

func (accessRepository) GroupAllowed(
	context.Context,
	security.UserID,
	permission.Code,
) (bool, error) {
	return false, nil
}

func (accessRepository) GuestAllowed(
	_ context.Context,
	code permission.Code,
) (bool, error) {
	return code == permission.MustCode(
		"core",
		"site",
		permission.Read,
	) || code == permission.MustCode(
		"core",
		"resource",
		permission.Read,
	), nil
}

func (accessRepository) GuestPermissions(
	context.Context,
) ([]coreaccess.Grant, error) {
	return nil, nil
}

func (accessRepository) GrantGuest(
	context.Context,
	*security.UserID,
	permission.Code,
) (coreaccess.Grant, error) {
	return coreaccess.Grant{}, nil
}

func (accessRepository) RevokeGuest(
	context.Context,
	permission.Code,
) error {
	return nil
}

type deniedAccessRepository struct {
	accessRepository
}

func (deniedAccessRepository) GuestAllowed(
	context.Context,
	permission.Code,
) (bool, error) {
	return false, nil
}

type privilegedUserAccessRepository struct {
	accessRepository
}

func (privilegedUserAccessRepository) Subject(
	context.Context,
	security.UserID,
) (coreaccess.Subject, error) {
	return coreaccess.Subject{
		Exists:  true,
		Active:  true,
		IsSuper: true,
	}, nil
}

type resourceRepository struct {
	byPath map[string]resource.Resource
	byID   map[resource.ID]resource.Resource
}

func (resourceRepository) Create(
	context.Context,
	*security.UserID,
	resource.Resource,
	resource.ValidateImageMedia,
) (resource.Resource, error) {
	return resource.Resource{}, resource.ErrNotFound
}

func (r resourceRepository) ByID(
	_ context.Context,
	id resource.ID,
) (resource.Resource, error) {
	item, exists := r.byID[id]
	if exists {
		return resource.Clone(item), nil
	}
	return resource.Resource{}, resource.ErrNotFound
}

func (r resourceRepository) ByPath(
	_ context.Context,
	siteID site.ID,
	path string,
) (resource.Resource, error) {
	item, exists := r.byPath[path]
	if exists && item.SiteID == siteID {
		return resource.Clone(item), nil
	}
	return resource.Resource{}, resource.ErrNotFound
}

func (resourceRepository) ListBySite(
	context.Context,
	site.ID,
) ([]resource.Resource, error) {
	return nil, nil
}

func (resourceRepository) Update(
	context.Context,
	*security.UserID,
	resource.Resource,
	resource.Resource,
	resource.ValidateImageMedia,
) (resource.Resource, error) {
	return resource.Resource{}, resource.ErrNotFound
}

func (resourceRepository) Delete(
	context.Context,
	resource.ID,
) error {
	return resource.ErrNotFound
}

type databaseFactory struct {
	files     corefile.Repository
	sites     site.Repository
	resources resource.Repository
	access    coreaccess.Repository
}

func (databaseFactory) ModuleCode() kernel.ModuleCode { return core.ModuleCode }
func (f databaseFactory) Build(kernel.DBConnector) (kernel.ModuleDatabase, error) {
	return database{
		files:     f.files,
		sites:     f.sites,
		resources: f.resources,
		access:    f.access,
	}, nil
}

type fileRepository struct {
	item corefile.File
}

func (fileRepository) NameAvailable(
	context.Context,
	filesystem.Code,
	*corefile.FolderID,
	string,
) error {
	return nil
}
func (fileRepository) CreateFolder(
	context.Context,
	corefile.Folder,
) (corefile.Folder, error) {
	return corefile.Folder{}, corefile.ErrFolderNotFound
}
func (fileRepository) FolderByID(
	context.Context,
	corefile.FolderID,
) (corefile.Folder, error) {
	return corefile.Folder{}, corefile.ErrFolderNotFound
}
func (fileRepository) ListFolders(
	context.Context,
	filesystem.Code,
	*corefile.FolderID,
) ([]corefile.Folder, error) {
	return nil, nil
}
func (fileRepository) CreateFile(
	context.Context,
	corefile.File,
) (corefile.File, error) {
	return corefile.File{}, corefile.ErrNotFound
}
func (r fileRepository) FileByID(
	_ context.Context,
	id corefile.ID,
) (corefile.File, error) {
	if r.item.ID != id {
		return corefile.File{}, corefile.ErrNotFound
	}
	return r.item, nil
}
func (fileRepository) ListFiles(
	context.Context,
	filesystem.Code,
	*corefile.FolderID,
) ([]corefile.File, error) {
	return nil, nil
}
func (fileRepository) MoveFile(
	context.Context,
	*security.UserID,
	corefile.ID,
	*corefile.FolderID,
) (corefile.File, error) {
	return corefile.File{}, corefile.ErrNotFound
}
func (fileRepository) MoveFolder(
	context.Context,
	*security.UserID,
	corefile.FolderID,
	*corefile.FolderID,
) (corefile.Folder, error) {
	return corefile.Folder{}, corefile.ErrFolderNotFound
}
func (fileRepository) DeleteFile(
	context.Context,
	corefile.ID,
	corefile.DeletePhysical,
) error {
	return corefile.ErrNotFound
}
func (fileRepository) DeleteFolder(
	context.Context,
	corefile.FolderID,
	corefile.DeletePhysical,
) error {
	return corefile.ErrFolderNotFound
}

func TestHandlerLooksUpCompiledRuntimeByRequestHost(t *testing.T) {
	runtimeApp, err := appkernel.New(context.Background(), appkernel.Definition{
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: connectorFactory{},
			Adapters:  []kernel.ModuleDatabaseFactory{databaseFactory{}},
		},
		Profiles: []kernel.Profile{{
			Code: "dev",
			Modules: []kernel.ProfileModule{
				{Module: core.Module{}},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = runtimeApp.Close() }()
	if err := runtimeApp.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}

	handler, err := httpserver.NewHandler(runtimeApp)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/_cms/runtime", nil)
	request.Host = "EXAMPLE.COM.:8080"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %q", response.Code, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodGet, "/_cms/runtime", nil)
	request.Host = "missing.example.com"
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("unknown host status = %d", response.Code)
	}
}

func TestHandlerHidesRuntimeWithoutGuestPermissionOrPublicFlag(
	t *testing.T,
) {
	tests := []struct {
		name    string
		factory databaseFactory
	}{
		{
			name: "public site without guest grant",
			factory: databaseFactory{
				sites:  repository{isPublic: true},
				access: deniedAccessRepository{},
			},
		},
		{
			name: "private site with guest grant",
			factory: databaseFactory{
				sites: repository{isPublic: false},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			runtimeApp, err := appkernel.New(
				context.Background(),
				appkernel.Definition{
					MainDatabase: appkernel.DatabaseDefinition{
						Connector: connectorFactory{},
						Adapters: []kernel.ModuleDatabaseFactory{
							test.factory,
						},
					},
					Profiles: []kernel.Profile{{
						Code: "dev",
						Modules: []kernel.ProfileModule{
							{Module: core.Module{}},
						},
					}},
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = runtimeApp.Close() }()
			if err := runtimeApp.Boot(context.Background()); err != nil {
				t.Fatal(err)
			}
			handler, err := httpserver.NewHandler(runtimeApp)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(
				http.MethodGet,
				"/_cms/runtime",
				nil,
			)
			request.Host = "example.com"
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf(
					"status = %d, body = %q",
					response.Code,
					response.Body.String(),
				)
			}
		})
	}
}

func TestHandlerDeliversPublicAndSignedPrivateLocalFiles(t *testing.T) {
	tests := []struct {
		name       string
		code       filesystem.Code
		visibility filesystem.Visibility
		signingKey string
	}{
		{
			name:       "public",
			code:       "public",
			visibility: filesystem.VisibilityPublic,
		},
		{
			name:       "private",
			code:       "private",
			visibility: filesystem.VisibilityPrivate,
			signingKey: strings.Repeat("private-key", 4),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			key := "objects/2026/test"
			target := filepath.Join(root, filepath.FromSlash(key))
			if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
				t.Fatal(err)
			}
			content := "delivered content"
			if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}

			repository := fileRepository{item: corefile.File{
				ID:       1,
				Storage:  test.code,
				Name:     "hello.txt",
				MIMEType: "text/plain; charset=utf-8",
				Size:     int64(len(content)),
				Path:     key,
			}}
			runtimeApp, err := appkernel.New(
				context.Background(),
				appkernel.Definition{
					MainDatabase: appkernel.DatabaseDefinition{
						Connector: connectorFactory{},
						Adapters: []kernel.ModuleDatabaseFactory{
							databaseFactory{files: repository},
						},
					},
					Filesystems: []filesystem.Factory{
						localstorage.Factory{Config: localstorage.Config{
							Code:       test.code,
							Visibility: test.visibility,
							Root:       root,
							BaseURL:    "https://cms.example.test",
							SigningKey: test.signingKey,
						}},
					},
					Profiles: []kernel.Profile{{
						Code: "dev",
						Modules: []kernel.ProfileModule{
							{Module: core.Module{}},
						},
					}},
				},
			)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = runtimeApp.Close() }()
			if err := runtimeApp.Boot(context.Background()); err != nil {
				t.Fatal(err)
			}

			rawURL := "https://cms.example.test/_cms/files/1"
			if test.visibility == filesystem.VisibilityPrivate {
				rawURL, err = runtimeApp.TemporaryFileURL(
					context.Background(),
					security.System(),
					1,
					time.Now().Add(time.Hour),
				)
				if err != nil {
					t.Fatal(err)
				}
			}
			handler, err := httpserver.NewHandler(runtimeApp)
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodGet, rawURL, nil)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusOK ||
				response.Body.String() != content ||
				response.Header().Get("Content-Type") !=
					"text/plain; charset=utf-8" {
				t.Fatalf(
					"response = %d, %q, %#v",
					response.Code,
					response.Body.String(),
					response.Header(),
				)
			}

			if test.visibility == filesystem.VisibilityPrivate {
				request = httptest.NewRequest(
					http.MethodGet,
					rawURL+"tampered",
					nil,
				)
				response = httptest.NewRecorder()
				handler.ServeHTTP(response, request)
				if response.Code != http.StatusNotFound {
					t.Fatalf("tampered URL status = %d", response.Code)
				}
			}
		})
	}
}

type transportResourceType struct {
	code resourcetype.Code
}

func (t transportResourceType) Code() resourcetype.Code {
	return t.code
}

func (transportResourceType) PathMode() resourcetype.PathMode {
	return resourcetype.PathRoute
}

func (transportResourceType) Normalize(
	payload resourcetype.Payload,
) (resourcetype.Payload, error) {
	return payload, nil
}

type transportModule struct {
	code         kernel.ModuleCode
	resourceType resourcetype.Type
	order        *[]string
}

func (m transportModule) Code() kernel.ModuleCode {
	return m.code
}

func (m transportModule) Registry() kernel.ModuleRegistry {
	return kernel.ModuleRegistry{
		ResourceTypes: []resourcetype.Type{m.resourceType},
	}
}

func (m transportModule) Build(
	context.Context,
	kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	return &transportRuntime{
		code:         m.code,
		resourceType: m.resourceType.Code(),
		order:        m.order,
	}, nil
}

type transportRuntime struct {
	code         kernel.ModuleCode
	resourceType resourcetype.Code
	order        *[]string
}

func (r *transportRuntime) ModuleCode() kernel.ModuleCode {
	return r.code
}

func (r *transportRuntime) HTTP() httptransport.Builder {
	return httptransport.BuilderFunc(func(
		context.Context,
	) (httptransport.Contribution, error) {
		return httptransport.Contribution{
			Middleware: []httptransport.MiddlewareDefinition{
				{
					Code:       "transport.profile",
					Scope:      httptransport.MiddlewareProfile,
					Middleware: recordHTTPOrder(r.order, "profile"),
				},
				{
					Code:       "transport.module",
					Scope:      httptransport.MiddlewareModule,
					Middleware: recordHTTPOrder(r.order, "module"),
				},
				{
					Code:       "transport.route",
					Scope:      httptransport.MiddlewareLocal,
					Middleware: recordHTTPOrder(r.order, "route"),
				},
				{
					Code:       "transport.resource",
					Scope:      httptransport.MiddlewareLocal,
					Middleware: recordHTTPOrder(r.order, "resource"),
				},
			},
			Routes: func(registrar httptransport.Registrar) error {
				return registrar.Route(httptransport.Route{
					Name:    "transport.custom",
					Method:  http.MethodGet,
					Pattern: "/custom",
					Handler: http.HandlerFunc(func(
						response http.ResponseWriter,
						_ *http.Request,
					) {
						*r.order = append(*r.order, "handler")
						response.WriteHeader(http.StatusNoContent)
					}),
					Middleware: []httptransport.MiddlewareCode{
						"transport.route",
					},
				})
			},
			ResourceHandlers: []httptransport.ResourceHandler{{
				Type: r.resourceType,
				Handler: http.HandlerFunc(func(
					response http.ResponseWriter,
					_ *http.Request,
				) {
					*r.order = append(*r.order, "handler")
					response.WriteHeader(http.StatusNoContent)
				}),
				Middleware: []httptransport.MiddlewareCode{
					"transport.resource",
				},
			}},
		}, nil
	})
}

func recordHTTPOrder(
	order *[]string,
	marker string,
) httptransport.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			response http.ResponseWriter,
			request *http.Request,
		) {
			*order = append(*order, marker)
			next.ServeHTTP(response, request)
		})
	}
}

func stringPointer(value string) *string {
	return &value
}

func newTransportTestApp(
	t *testing.T,
	module transportModule,
	resources resource.Repository,
) *appkernel.App {
	t.Helper()
	runtimeApp, err := appkernel.New(
		context.Background(),
		appkernel.Definition{
			MainDatabase: appkernel.DatabaseDefinition{
				Connector: connectorFactory{},
				Adapters: []kernel.ModuleDatabaseFactory{
					databaseFactory{resources: resources},
				},
			},
			Profiles: []kernel.Profile{{
				Code: "dev",
				Modules: []kernel.ProfileModule{
					{Module: core.Module{}},
					{Module: module},
				},
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = runtimeApp.Close() })
	if err := runtimeApp.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	return runtimeApp
}

func TestHandlerMiddlewareOrderForRoutesAndResources(t *testing.T) {
	order := make([]string, 0, 5)
	resourceType := transportResourceType{code: "transport_test"}
	module := transportModule{
		code:         "transport",
		resourceType: resourceType,
		order:        &order,
	}
	runtimeApp := newTransportTestApp(
		t,
		module,
		resourceRepository{
			byPath: map[string]resource.Resource{
				"/article": {
					ID:       1,
					SiteID:   1,
					Type:     resourceType.Code(),
					Path:     stringPointer("/article"),
					IsPublic: true,
				},
			},
		},
	)
	handler, err := httpserver.NewHandler(
		runtimeApp,
		httpserver.WithPlatformMiddleware(
			recordHTTPOrder(&order, "platform"),
		),
	)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/custom", nil)
	request.Host = "example.com"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent ||
		strings.Join(order, ",") !=
			"platform,profile,module,route,handler" {
		t.Fatalf("custom response = %d, order = %v", response.Code, order)
	}

	order = order[:0]
	request = httptest.NewRequest(http.MethodGet, "/article", nil)
	request.Host = "example.com"
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent ||
		strings.Join(order, ",") !=
			"platform,profile,module,resource,handler" {
		t.Fatalf("resource response = %d, order = %v", response.Code, order)
	}
}

func TestResourceRootTrailingSlashAndQueryPolicy(t *testing.T) {
	order := make([]string, 0)
	resourceType := transportResourceType{code: "route_policy"}
	module := transportModule{
		code:         "transport",
		resourceType: resourceType,
		order:        &order,
	}
	repository := resourceRepository{
		byPath: map[string]resource.Resource{
			"/": {
				ID:       1,
				SiteID:   1,
				Type:     resourceType.Code(),
				Path:     stringPointer("/"),
				IsPublic: true,
			},
			"/section": {
				ID:       2,
				SiteID:   1,
				Type:     resourceType.Code(),
				Path:     stringPointer("/section"),
				IsPublic: true,
			},
			"/_cms/claimed": {
				ID:       3,
				SiteID:   1,
				Type:     resourceType.Code(),
				Path:     stringPointer("/_cms/claimed"),
				IsPublic: true,
			},
		},
	}
	handler, err := httpserver.NewHandler(
		newTransportTestApp(t, module, repository),
	)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path string
		want int
	}{
		{path: "/", want: http.StatusNoContent},
		{path: "/section", want: http.StatusNoContent},
		{path: "/section?from=query", want: http.StatusNoContent},
		{path: "/section/", want: http.StatusNotFound},
		{path: "/_cms/claimed", want: http.StatusNotFound},
	}
	for _, test := range tests {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		request.Host = "example.com"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != test.want {
			t.Fatalf(
				"path %q status = %d, want %d",
				test.path,
				response.Code,
				test.want,
			)
		}
	}
}

func TestPlatformRuntimeMethodMismatchKeeps405AndAllow(t *testing.T) {
	runtimeApp, err := appkernel.New(context.Background(), appkernel.Definition{
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: connectorFactory{},
			Adapters:  []kernel.ModuleDatabaseFactory{databaseFactory{}},
		},
		Profiles: []kernel.Profile{{
			Code: "dev",
			Modules: []kernel.ProfileModule{
				{Module: core.Module{}},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = runtimeApp.Close() }()
	if err := runtimeApp.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	handler, err := httpserver.NewHandler(runtimeApp)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "/_cms/runtime", nil)
	request.Host = "example.com"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusMethodNotAllowed ||
		!strings.Contains(response.Header().Get("Allow"), http.MethodGet) {
		t.Fatalf(
			"status = %d, Allow = %q",
			response.Code,
			response.Header().Get("Allow"),
		)
	}
}

func TestPlatformAuthenticationActorPrecedesPrivateSiteResolution(
	t *testing.T,
) {
	runtimeApp, err := appkernel.New(context.Background(), appkernel.Definition{
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: connectorFactory{},
			Adapters: []kernel.ModuleDatabaseFactory{databaseFactory{
				sites:  repository{isPublic: false},
				access: privilegedUserAccessRepository{},
			}},
		},
		Profiles: []kernel.Profile{{
			Code: "dev",
			Modules: []kernel.ProfileModule{
				{Module: core.Module{}},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = runtimeApp.Close() }()
	if err := runtimeApp.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}

	handler, err := httpserver.NewHandler(
		runtimeApp,
		httpserver.WithPlatformMiddleware(
			func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(
					response http.ResponseWriter,
					request *http.Request,
				) {
					next.ServeHTTP(
						response,
						request.WithContext(
							httptransport.WithActor(
								request.Context(),
								security.User(42),
							),
						),
					)
				})
			},
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/_cms/runtime", nil)
	request.Host = "example.com"
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, body = %q",
			response.Code,
			response.Body.String(),
		)
	}
}

func TestStandardLinkResourceHandlersRedirect(t *testing.T) {
	externalURL := "https://example.org/article"
	externalPath := "/external"
	shortcutPath := "/shortcut"
	targetPath := "/target"
	targetID := resource.ID(3)
	resources := resourceRepository{
		byPath: map[string]resource.Resource{
			externalPath: {
				ID:          1,
				SiteID:      1,
				Type:        resourcetype.Link,
				Path:        &externalPath,
				ExternalURL: &externalURL,
				IsPublic:    true,
			},
			shortcutPath: {
				ID:               2,
				SiteID:           1,
				Type:             resourcetype.ResourceLink,
				Path:             &shortcutPath,
				TargetResourceID: &targetID,
				IsPublic:         true,
			},
		},
		byID: map[resource.ID]resource.Resource{
			targetID: {
				ID:       targetID,
				SiteID:   1,
				Type:     resourcetype.Page,
				Path:     &targetPath,
				IsPublic: true,
			},
		},
	}
	runtimeApp, err := appkernel.New(context.Background(), appkernel.Definition{
		MainDatabase: appkernel.DatabaseDefinition{
			Connector: connectorFactory{},
			Adapters: []kernel.ModuleDatabaseFactory{
				databaseFactory{resources: resources},
			},
		},
		Profiles: []kernel.Profile{{
			Code: "dev",
			Modules: []kernel.ProfileModule{
				{Module: core.Module{}},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = runtimeApp.Close() }()
	if err := runtimeApp.Boot(context.Background()); err != nil {
		t.Fatal(err)
	}
	handler, err := httpserver.NewHandler(runtimeApp)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path     string
		location string
	}{
		{path: externalPath, location: externalURL},
		{path: shortcutPath, location: targetPath},
	}
	for _, test := range tests {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		request.Host = "example.com"
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusFound ||
			response.Header().Get("Location") != test.location {
			t.Fatalf(
				"path %q response = %d, Location = %q",
				test.path,
				response.Code,
				response.Header().Get("Location"),
			)
		}
	}
}
