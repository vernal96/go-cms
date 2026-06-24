package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vernal96/go-cms/core"
)

func TestWriteHandlerResultHTTPResponse(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeHandlerResult(context.Background(), recorder, core.HTTPResponse{
		StatusCode:  http.StatusCreated,
		ContentType: "text/html; charset=utf-8",
		Body:        []byte("<h1>Hello</h1>"),
	})

	if recorder.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf(
			"unexpected content type: %q",
			recorder.Header().Get("Content-Type"),
		)
	}
	if recorder.Body.String() != "<h1>Hello</h1>" {
		t.Fatalf("unexpected body: %q", recorder.Body.String())
	}
}

func TestWriteHandlerResultJSONFallback(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeHandlerResult(context.Background(), recorder, map[string]any{
		"ok": true,
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}
	if recorder.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf(
			"unexpected content type: %q",
			recorder.Header().Get("Content-Type"),
		)
	}
	if recorder.Body.String() != "{\"ok\":true}\n" {
		t.Fatalf("unexpected body: %q", recorder.Body.String())
	}
}

func TestFindRouteAndFallbackRoute(t *testing.T) {
	runtime := newRoutingRuntime(t)

	t.Run("exact CMS route", func(t *testing.T) {
		route, exists := findRoute(
			runtime,
			http.MethodGet,
			"/_cms/resource",
		)
		if !exists || route.Path != "/_cms/resource" {
			t.Fatalf("unexpected route: %#v, exists=%t", route, exists)
		}
	})

	t.Run("page fallback", func(t *testing.T) {
		route, exists := findFallbackRoute(
			runtime,
			http.MethodGet,
			"/about",
		)
		if !exists || route.Path != "/" {
			t.Fatalf("unexpected route: %#v, exists=%t", route, exists)
		}
	})

	t.Run("CMS path does not fallback", func(t *testing.T) {
		if _, exists := findFallbackRoute(
			runtime,
			http.MethodGet,
			"/_cms/unknown",
		); exists {
			t.Fatal("CMS path must not use page fallback")
		}
	})

	t.Run("POST does not fallback", func(t *testing.T) {
		if _, exists := findFallbackRoute(
			runtime,
			http.MethodPost,
			"/about",
		); exists {
			t.Fatal("POST path must not use page fallback")
		}
	})
}

func newRoutingRuntime(t *testing.T) *core.SiteRuntime {
	t.Helper()

	app, err := core.NewApp(
		routingCacheManager{},
		routingFileStorageManager{},
		core.NullEventBus{},
		core.NullLogger{},
		routingResourceRepository{},
		routingResourceFieldValueRepository{},
	)
	if err != nil {
		t.Fatal(err)
	}

	registry := core.NewRuntimeRegistry()
	if err := registry.Controllers().Register(routingController{}); err != nil {
		t.Fatal(err)
	}

	runtime, err := core.NewSiteRuntime(
		app,
		core.Site{
			ID:          1,
			ProfileCode: "main",
		},
		routingSiteProfile{},
		registry,
	)
	if err != nil {
		t.Fatal(err)
	}

	return runtime
}

type routingController struct{}

func (routingController) Routes() []core.Route {
	handler := func(
		ctx context.Context,
		runtime *core.SiteRuntime,
		request *http.Request,
	) (any, error) {
		return nil, nil
	}

	return []core.Route{
		{
			Method:  core.RouteMethodGet,
			Path:    "/_cms/resource",
			Handler: handler,
		},
		{
			Method:  core.RouteMethodGet,
			Path:    "/",
			Handler: handler,
		},
	}
}

type routingSiteProfile struct{}

func (routingSiteProfile) Code() string {
	return "main"
}

func (routingSiteProfile) Modules() []core.Module {
	return nil
}

type routingCacheManager struct{}

func (routingCacheManager) Store(
	name core.CacheStoreName,
) (core.CacheStore, error) {
	return core.NullCacheStore{}, nil
}

func (routingCacheManager) Scope(
	scope core.CacheScope,
) (core.CacheStore, error) {
	return core.NullCacheStore{}, nil
}

type routingFileStorageManager struct{}

func (routingFileStorageManager) Disk(
	name core.FileDisk,
) (core.FileStorage, error) {
	return core.NullFileStorage{}, nil
}

type routingResourceRepository struct{}

func (routingResourceRepository) FindByID(
	ctx context.Context,
	id core.ResourceID,
) (core.Resource, error) {
	return core.Resource{}, nil
}

func (routingResourceRepository) FindByPath(
	ctx context.Context,
	siteID int64,
	path string,
) (core.Resource, error) {
	return core.Resource{}, nil
}

func (routingResourceRepository) FindChildren(
	ctx context.Context,
	parentID core.ResourceID,
) ([]core.Resource, error) {
	return nil, nil
}

type routingResourceFieldValueRepository struct{}

func (routingResourceFieldValueRepository) FindByResourceID(
	ctx context.Context,
	resourceID core.ResourceID,
) ([]core.ResourceFieldValue, error) {
	return nil, nil
}

func (routingResourceFieldValueRepository) FindByResourceAndField(
	ctx context.Context,
	resourceID core.ResourceID,
	field core.ResourceFieldCode,
) (core.ResourceFieldValue, error) {
	return core.ResourceFieldValue{}, nil
}

func (routingResourceFieldValueRepository) Save(
	ctx context.Context,
	value core.ResourceFieldValue,
) (core.ResourceFieldValue, error) {
	return value, nil
}
