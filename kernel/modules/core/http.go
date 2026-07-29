package core

import (
	"context"
	"errors"
	"net/http"

	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/security"
	httptransport "github.com/vernal96/go-cms/kernel/transport/http"
)

func (r *Runtime) HTTP() httptransport.Builder {
	return httptransport.BuilderFunc(func(
		context.Context,
	) (httptransport.Contribution, error) {
		if r == nil {
			return httptransport.Contribution{}, errors.New(
				"core runtime is nil",
			)
		}

		resolverOptions := make([]resource.RouteResolverOption, 0, 1)
		if r.resourcePreview != nil {
			resolverOptions = append(
				resolverOptions,
				resource.WithPreviewPolicy(r.resourcePreview),
			)
		}
		resolver, err := resource.NewRouteResolver(
			r.database.Resources(),
			r.authorization,
			resolverOptions...,
		)
		if err != nil {
			return httptransport.Contribution{}, err
		}

		return httptransport.Contribution{
			ResourceHandlers: []httptransport.ResourceHandler{
				{
					Type: resourcetype.Page,
					Handler: http.HandlerFunc(
						unavailablePageRenderer,
					),
				},
				{
					Type:    resourcetype.Link,
					Handler: externalLinkHandler{},
				},
				{
					Type: resourcetype.ResourceLink,
					Handler: resourceLinkHandler{
						resolver: resolver,
					},
				},
			},
			TerminalResource: &httptransport.TerminalResourceHandler{
				Factory: func(
					handlers httptransport.ResourceHandlers,
				) (http.Handler, error) {
					if handlers == nil {
						return nil, errors.New(
							"core resource handlers are nil",
						)
					}
					return &terminalResourceHandler{
						resolver: resolver,
						handlers: handlers,
					}, nil
				},
			},
		}, nil
	})
}

type terminalResourceHandler struct {
	resolver *resource.RouteResolver
	handlers httptransport.ResourceHandlers
}

func (h *terminalResourceHandler) ServeHTTP(
	response http.ResponseWriter,
	request *http.Request,
) {
	siteRuntime, siteExists := httptransport.SiteRuntimeFromContext(
		request.Context(),
	)
	actor, actorExists := httptransport.ActorFromContext(request.Context())
	if !siteExists || !actorExists {
		http.Error(
			response,
			"resource request context is incomplete",
			http.StatusInternalServerError,
		)
		return
	}

	path, err := resource.NormalizeLookupPath(request.URL.Path)
	if err != nil {
		http.NotFound(response, request)
		return
	}

	item, err := h.resolver.ResolvePublishedByPath(
		request.Context(),
		actor,
		siteRuntime,
		path,
		resource.ResolveRouteOptions{
			Preview: httptransport.PreviewFromContext(request.Context()),
		},
	)
	if err != nil {
		writeResourceRouteError(response, request, err)
		return
	}

	handler, exists := h.handlers.Handler(item.Type)
	if !exists {
		http.NotFound(response, request)
		return
	}

	handler.ServeHTTP(
		response,
		request.WithContext(
			httptransport.WithResource(request.Context(), item),
		),
	)
}

// Page rendering is deliberately a transport integration point. The project
// does not yet contain a template engine, so the registered standard handler
// preserves the existing public 404 instead of inventing a response format.
func unavailablePageRenderer(
	response http.ResponseWriter,
	request *http.Request,
) {
	http.NotFound(response, request)
}

type externalLinkHandler struct{}

func (externalLinkHandler) ServeHTTP(
	response http.ResponseWriter,
	request *http.Request,
) {
	item, exists := httptransport.ResourceFromContext(request.Context())
	if !exists || item.ExternalURL == nil {
		http.NotFound(response, request)
		return
	}
	http.Redirect(
		response,
		request,
		*item.ExternalURL,
		http.StatusFound,
	)
}

type resourceLinkHandler struct {
	resolver *resource.RouteResolver
}

func (h resourceLinkHandler) ServeHTTP(
	response http.ResponseWriter,
	request *http.Request,
) {
	item, resourceExists := httptransport.ResourceFromContext(
		request.Context(),
	)
	siteRuntime, siteExists := httptransport.SiteRuntimeFromContext(
		request.Context(),
	)
	actor, actorExists := httptransport.ActorFromContext(request.Context())
	if !resourceExists ||
		!siteExists ||
		!actorExists ||
		item.TargetResourceID == nil {
		http.NotFound(response, request)
		return
	}

	target, err := h.resolver.ResolvePublishedByID(
		request.Context(),
		actor,
		siteRuntime,
		*item.TargetResourceID,
		resource.ResolveRouteOptions{
			Preview: httptransport.PreviewFromContext(request.Context()),
		},
	)
	if err != nil {
		writeResourceRouteError(response, request, err)
		return
	}
	if target.Path == nil {
		http.NotFound(response, request)
		return
	}
	http.Redirect(
		response,
		request,
		*target.Path,
		http.StatusFound,
	)
}

func writeResourceRouteError(
	response http.ResponseWriter,
	request *http.Request,
	err error,
) {
	switch {
	case errors.Is(err, resource.ErrNotFound),
		errors.Is(err, security.ErrForbidden),
		errors.Is(err, security.ErrUnauthenticated):
		http.NotFound(response, request)
	default:
		http.Error(
			response,
			"resource route failed",
			http.StatusInternalServerError,
		)
	}
}

var _ httptransport.Provider = (*Runtime)(nil)
