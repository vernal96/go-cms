package core

import (
	"context"
	"net/http"
)

type RouteMethod string

const (
	RouteMethodGet    RouteMethod = http.MethodGet
	RouteMethodPost   RouteMethod = http.MethodPost
	RouteMethodPut    RouteMethod = http.MethodPut
	RouteMethodPatch  RouteMethod = http.MethodPatch
	RouteMethodDelete RouteMethod = http.MethodDelete
)

type Route struct {
	Method  RouteMethod
	Path    string
	Handler ControllerHandler
}

type ControllerHandler func(ctx context.Context, runtime *SiteRuntime, request *http.Request) (any, error)

type Controller interface {
	Routes() []Route
}
