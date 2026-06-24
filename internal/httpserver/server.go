package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/vernal96/go-cms/core"
)

type Server struct {
	sites          core.SiteRepository
	runtimeFactory *core.SiteRuntimeFactory
}

func New(sites core.SiteRepository, runtimeFactory *core.SiteRuntimeFactory) (*Server, error) {
	if sites == nil {
		return nil, errors.New("site repository is nil")
	}

	if runtimeFactory == nil {
		return nil, errors.New("site runtime factory is nil")
	}

	return &Server{
		sites:          sites,
		runtimeFactory: runtimeFactory,
	}, nil
}

func (s *Server) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	domain := hostWithoutPort(request.Host)

	site, err := s.sites.FindByDomain(ctx, domain)
	if err != nil {
		if errors.Is(err, core.ErrSiteNotFound) {
			http.Error(response, "site not found", http.StatusNotFound)
			return
		}

		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	runtime, err := s.runtimeFactory.Make(ctx, site)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	route, exists := findRoute(runtime, request.Method, request.URL.Path)
	if !exists {
		http.Error(response, "route not found", http.StatusNotFound)
		return
	}

	result, err := route.Handler(ctx, runtime, request)
	if err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(ctx, response, result)
}

func findRoute(runtime *core.SiteRuntime, method string, path string) (core.Route, bool) {
	if runtime == nil {
		return core.Route{}, false
	}

	for _, controller := range runtime.Registry().Controllers().All() {
		for _, route := range controller.Routes() {
			if string(route.Method) == method && route.Path == path {
				return route, true
			}
		}
	}

	return core.Route{}, false
}

func hostWithoutPort(host string) string {
	if host == "" {
		return ""
	}

	domain, _, err := net.SplitHostPort(host)
	if err == nil {
		return domain
	}

	return host
}

func writeJSON(ctx context.Context, response http.ResponseWriter, value any) {
	if err := ctx.Err(); err != nil {
		http.Error(response, err.Error(), http.StatusRequestTimeout)
		return
	}

	payload, err := json.Marshal(value)
	if err != nil {
		http.Error(response, fmt.Sprintf("encode response: %v", err), http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = response.Write(append(payload, '\n'))
}
