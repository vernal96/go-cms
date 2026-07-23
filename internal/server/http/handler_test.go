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
	corefile "github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
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

type repository struct{}

func (repository) List(context.Context) ([]site.Site, error) {
	return []site.Site{
		{
			ID:          1,
			ProfileCode: "dev",
			Domain:      "example.com",
			Locale:      "ru-RU",
		},
	}, nil
}

func (repository) UpdateSettings(
	context.Context,
	site.ID,
	map[string]any,
) error {
	return nil
}

type database struct {
	files corefile.Repository
}

func (database) ModuleCode() kernel.ModuleCode { return core.ModuleCode }
func (database) Sites() site.Repository        { return repository{} }
func (database) Resources() resource.Repository {
	return resourceRepository{}
}
func (d database) Files() corefile.Repository {
	if d.files != nil {
		return d.files
	}
	return fileRepository{}
}

type resourceRepository struct{}

func (resourceRepository) Create(
	context.Context,
	resource.Resource,
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
	resource.Resource,
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
	files corefile.Repository
}

func (databaseFactory) ModuleCode() kernel.ModuleCode { return core.ModuleCode }
func (f databaseFactory) Build(kernel.DBConnector) (kernel.ModuleDatabase, error) {
	return database{files: f.files}, nil
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
	corefile.ID,
	*corefile.FolderID,
) (corefile.File, error) {
	return corefile.File{}, corefile.ErrNotFound
}
func (fileRepository) MoveFolder(
	context.Context,
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
