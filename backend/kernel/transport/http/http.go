package httptransport

import (
	"context"
	"net/http"

	"github.com/vernal96/go-cms/kernel/security"
)

type MiddlewareCode string
type ResourceHandlerCode string
type Middleware func(http.Handler) http.Handler

type MiddlewareScope uint8

const (
	MiddlewareProfile MiddlewareScope = iota + 1
	MiddlewareModule
	MiddlewareLocal
)

type MiddlewareDefinition struct {
	Code       MiddlewareCode
	Scope      MiddlewareScope
	Middleware Middleware
}

type Route struct {
	Name       string
	Method     string
	Pattern    string
	Handler    http.Handler
	Middleware []MiddlewareCode
}

type Mount struct {
	Name       string
	Pattern    string
	Handler    http.Handler
	Middleware []MiddlewareCode
}

type RegisterRoutes func(Registrar) error

// Registrar is intentionally narrower than a root HTTP router. In particular,
// modules cannot replace the profile NotFound or MethodNotAllowed handlers.
type Registrar interface {
	Route(Route) error
	Mount(Mount) error
	Group(
		prefix string,
		middleware []MiddlewareCode,
		register RegisterRoutes,
	) error
}

type ResourceHandler struct {
	Type       ResourceHandlerCode
	Handler    http.Handler
	Middleware []MiddlewareCode
}

type ResourceHandlers interface {
	Handler(ResourceHandlerCode) (http.Handler, bool)
}

type TerminalResourceHandler struct {
	Factory    func(ResourceHandlers) (http.Handler, error)
	Middleware []MiddlewareCode
}

type Contribution struct {
	Middleware       []MiddlewareDefinition
	Routes           RegisterRoutes
	ResourceHandlers []ResourceHandler
	TerminalResource *TerminalResourceHandler
}

// Builder is the module-owned controller construction boundary. Build creates
// controllers from runtime dependencies and returns routes with ready handlers.
type Builder interface {
	Build(context.Context) (Contribution, error)
}

type BuilderFunc func(context.Context) (Contribution, error)

func (f BuilderFunc) Build(ctx context.Context) (Contribution, error) {
	return f(ctx)
}

// Provider is optional. kernel.ModuleRuntime intentionally does not embed it.
type Provider interface {
	HTTP() Builder
}

type requestContextKey uint8

const (
	actorContextKey requestContextKey = iota + 1
	previewContextKey
)

func WithActor(ctx context.Context, actor security.Actor) context.Context {
	return context.WithValue(ctx, actorContextKey, actor)
}

func ActorFromContext(ctx context.Context) (security.Actor, bool) {
	if ctx == nil {
		return security.Actor{}, false
	}
	actor, exists := ctx.Value(actorContextKey).(security.Actor)
	return actor, exists
}

func WithPreview(ctx context.Context, preview bool) context.Context {
	return context.WithValue(ctx, previewContextKey, preview)
}

func PreviewFromContext(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	preview, _ := ctx.Value(previewContextKey).(bool)
	return preview
}
