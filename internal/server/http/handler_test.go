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
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	coreuser "github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
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
	files  corefile.Repository
	sites  site.Repository
	access coreaccess.Repository
}

func (database) ModuleCode() kernel.ModuleCode { return core.ModuleCode }
func (d database) Sites() site.Repository {
	if d.sites != nil {
		return d.sites
	}
	return repository{isPublic: true}
}
func (database) Resources() resource.Repository {
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

type resourceRepository struct{}

func (resourceRepository) Create(
	context.Context,
	*security.UserID,
	resource.Resource,
	resource.ValidateImageMedia,
) (resource.Resource, error) {
	return resource.Resource{}, resource.ErrNotFound
}

func (resourceRepository) ByID(
	context.Context,
	resource.ID,
) (resource.Resource, error) {
	return resource.Resource{}, resource.ErrNotFound
}

func (resourceRepository) ByPath(
	context.Context,
	site.ID,
	string,
) (resource.Resource, error) {
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
	files  corefile.Repository
	sites  site.Repository
	access coreaccess.Repository
}

func (databaseFactory) ModuleCode() kernel.ModuleCode { return core.ModuleCode }
func (f databaseFactory) Build(kernel.DBConnector) (kernel.ModuleDatabase, error) {
	return database{
		files:  f.files,
		sites:  f.sites,
		access: f.access,
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
