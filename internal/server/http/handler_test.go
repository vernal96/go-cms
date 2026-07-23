package httpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httpserver "github.com/vernal96/go-cms/internal/server/http"
	"github.com/vernal96/go-cms/kernel"
	appkernel "github.com/vernal96/go-cms/kernel/app"
	"github.com/vernal96/go-cms/kernel/modules/core"
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

type database struct{}

func (database) ModuleCode() kernel.ModuleCode { return core.ModuleCode }
func (database) Sites() site.Repository        { return repository{} }

type databaseFactory struct{}

func (databaseFactory) ModuleCode() kernel.ModuleCode { return core.ModuleCode }
func (databaseFactory) Build(kernel.DBConnector) (kernel.ModuleDatabase, error) {
	return database{}, nil
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
