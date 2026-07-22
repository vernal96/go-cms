package httpserver_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vernal96/go-cms/app"
	httpserver "github.com/vernal96/go-cms/internal/server/http"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

type connector struct{}

func (connector) Code() kernel.ConnectionCode { return "test" }
func (connector) Ping(context.Context) error  { return nil }
func (connector) Close() error                { return nil }

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

type database struct{}

func (database) ModuleCode() kernel.ModuleCode { return core.ModuleCode }
func (database) Sites() site.Repository        { return repository{} }

func TestHandlerLooksUpCompiledRuntimeByRequestHost(t *testing.T) {
	runtimeApp, err := app.New(context.Background(), app.AppConfig{
		MainDatabase: app.DatabaseBinding{
			Connector: connector{},
			Adapters:  []kernel.ModuleDatabase{database{}},
		},
	}, []kernel.Profile{
		{
			Code: "dev",
			Modules: []kernel.ProfileModule{
				{Module: core.Module{}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = runtimeApp.Close() }()

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
