package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
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
					Type:    resourcetype.Page,
					Handler: pageResourceHandler{logger: r.logger},
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

const (
	widgetUnavailableError = "widget_unavailable"
	invalidParamsError     = "invalid_params"
	instanceFailedError    = "instance_failed"
	renderFailedError      = "render_failed"
	invalidResultError     = "invalid_result"
)

type pageResourceResponse struct {
	Resource pageResourcePayload  `json:"resource"`
	Widgets  []pageWidgetResponse `json:"widgets"`
}

type pageResourcePayload struct {
	ID       resource.ID       `json:"id"`
	Type     resourcetype.Code `json:"type"`
	Template *template.Code    `json:"template"`
	Title    string            `json:"title"`
	Path     *string           `json:"path"`
	Content  string            `json:"content"`
}

type pageWidgetResponse struct {
	Code     widget.Code      `json:"code"`
	Position int              `json:"position"`
	Data     json.RawMessage  `json:"data,omitempty"`
	Error    *pageWidgetError `json:"error,omitempty"`
}

type pageWidgetError struct {
	Code string `json:"code"`
}

type pageResourceHandler struct {
	logger *slog.Logger
}

func (h pageResourceHandler) ServeHTTP(
	response http.ResponseWriter,
	request *http.Request,
) {
	ctx := request.Context()
	item, resourceExists := httptransport.ResourceFromContext(ctx)
	siteRuntime, siteExists := httptransport.SiteRuntimeFromContext(ctx)
	if !resourceExists || !siteExists {
		http.Error(
			response,
			"resource request context is incomplete",
			http.StatusInternalServerError,
		)
		return
	}

	result := pageResourceResponse{
		Resource: pageResourcePayload{
			ID:       item.ID,
			Type:     item.Type,
			Template: item.Template,
			Title:    item.Title,
			Path:     item.Path,
			Content:  item.Content,
		},
		Widgets: make([]pageWidgetResponse, 0, len(item.Widgets)),
	}

	for _, binding := range item.Widgets {
		if err := ctx.Err(); err != nil {
			return
		}

		rendered := pageWidgetResponse{
			Code:     binding.Code,
			Position: binding.Position,
		}
		runtime, exists := siteRuntime.Profile().Widget(binding.Code)
		if !exists {
			rendered.Error = h.widgetError(
				ctx,
				item,
				binding,
				widgetUnavailableError,
				fmt.Errorf("widget %q is unavailable", binding.Code),
			)
			result.Widgets = append(result.Widgets, rendered)
			continue
		}

		instance, err := runtime.New(binding.Params)
		if err != nil {
			code := instanceFailedError
			if errors.Is(err, widget.ErrInvalidParams) {
				code = invalidParamsError
			}
			rendered.Error = h.widgetError(
				ctx,
				item,
				binding,
				code,
				err,
			)
			result.Widgets = append(result.Widgets, rendered)
			continue
		}

		data, err := instance.Render(ctx, widget.RenderInput{
			Resource: widget.ResourceSnapshot{
				ID:      int64(item.ID),
				Content: item.Content,
			},
		})
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			rendered.Error = h.widgetError(
				ctx,
				item,
				binding,
				renderFailedError,
				err,
			)
			result.Widgets = append(result.Widgets, rendered)
			continue
		}
		if data == nil {
			rendered.Error = h.widgetError(
				ctx,
				item,
				binding,
				invalidResultError,
				errors.New("widget returned nil data"),
			)
			result.Widgets = append(result.Widgets, rendered)
			continue
		}
		rawData, err := json.Marshal(data)
		if err != nil {
			rendered.Error = h.widgetError(
				ctx,
				item,
				binding,
				invalidResultError,
				err,
			)
			result.Widgets = append(result.Widgets, rendered)
			continue
		}

		rendered.Data = rawData
		result.Widgets = append(result.Widgets, rendered)
	}

	raw, err := json.Marshal(result)
	if err != nil {
		h.logError(
			ctx,
			"resource response encoding failed",
			item,
			resource.WidgetBinding{},
			err,
		)
		http.Error(
			response,
			"resource response failed",
			http.StatusInternalServerError,
		)
		return
	}

	response.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	response.WriteHeader(http.StatusOK)
	_, _ = response.Write(append(raw, '\n'))
}

func (h pageResourceHandler) widgetError(
	ctx context.Context,
	item resource.Resource,
	binding resource.WidgetBinding,
	code string,
	err error,
) *pageWidgetError {
	h.logError(ctx, "resource widget failed", item, binding, err)
	return &pageWidgetError{Code: code}
}

func (h pageResourceHandler) logError(
	ctx context.Context,
	message string,
	item resource.Resource,
	binding resource.WidgetBinding,
	err error,
) {
	if h.logger == nil {
		return
	}

	attributes := []any{
		slog.String("event", "resource.widget.failed"),
		slog.Int64("resource.id", int64(item.ID)),
		slog.String("widget.code", string(binding.Code)),
		slog.Int("widget.position", binding.Position),
		slog.Any("error", err),
	}
	h.logger.ErrorContext(ctx, message, attributes...)
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
