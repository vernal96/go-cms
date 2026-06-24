package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

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
		route, exists = findFallbackRoute(runtime, request.Method, request.URL.Path)
	}
	if !exists {
		http.Error(response, "route not found", http.StatusNotFound)
		return
	}

	result, err := route.Handler(ctx, runtime, request)
	if err != nil {
		if errors.Is(err, core.ErrResourceNotFound) {
			http.Error(response, "resource not found", http.StatusNotFound)
			return
		}

		http.Error(response, err.Error(), http.StatusInternalServerError)
		return
	}

	writeHandlerResult(ctx, response, result)
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

func findFallbackRoute(
	runtime *core.SiteRuntime,
	method string,
	path string,
) (core.Route, bool) {
	if method != http.MethodGet {
		return core.Route{}, false
	}

	if strings.HasPrefix(path, "/_cms/") {
		return core.Route{}, false
	}

	return findRoute(runtime, http.MethodGet, "/")
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

func writeHandlerResult(
	ctx context.Context,
	response http.ResponseWriter,
	result any,
) {
	if err := ctx.Err(); err != nil {
		http.Error(response, err.Error(), http.StatusRequestTimeout)
		return
	}

	if httpResponse, ok := result.(core.HTTPResponse); ok {
		statusCode := httpResponse.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}

		contentType := httpResponse.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		response.Header().Set("Content-Type", contentType)
		response.WriteHeader(statusCode)
		_, _ = response.Write(httpResponse.Body)

		return
	}

	writeJSON(response, result)
}

func writeJSON(response http.ResponseWriter, value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		http.Error(response, fmt.Sprintf("encode response: %v", err), http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = response.Write(append(payload, '\n'))
}
