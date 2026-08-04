package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/app"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	corefile "github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/security"
	httptransport "github.com/vernal96/go-cms/kernel/transport/http"
)

type Handler struct {
	app      *app.App
	root     http.Handler
	profiles map[kernel.ProfileCode]http.Handler
}

type Option func(*handlerOptions) error

type handlerOptions struct {
	platformMiddleware []httptransport.Middleware
	accessTokens       security.AccessTokens
}

type runtimeResponse struct {
	SiteID      site.ID            `json:"site_id"`
	Domain      string             `json:"domain"`
	Locale      string             `json:"locale"`
	ProfileCode kernel.ProfileCode `json:"profile_code"`
	Settings    map[string]any     `json:"settings"`
}

func WithPlatformMiddleware(
	middleware ...httptransport.Middleware,
) Option {
	return func(options *handlerOptions) error {
		for index, current := range middleware {
			if current == nil {
				return errors.New(
					"platform middleware at index " +
						strconv.Itoa(index) + " is nil",
				)
			}
		}
		options.platformMiddleware = append(
			options.platformMiddleware,
			middleware...,
		)
		return nil
	}
}

func WithAccessTokens(tokens security.AccessTokens) Option {
	return func(options *handlerOptions) error {
		if isNilHTTPValue(tokens) {
			return errors.New("HTTP access token service is nil")
		}
		options.accessTokens = tokens
		return nil
	}
}

func NewHandler(
	application *app.App,
	options ...Option,
) (*Handler, error) {
	if application == nil {
		return nil, errors.New("app is nil")
	}

	config := handlerOptions{}
	for index, option := range options {
		if option == nil {
			return nil, errors.New(
				"HTTP handler option at index " +
					strconv.Itoa(index) + " is nil",
			)
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	if config.accessTokens == nil {
		return nil, errors.New("HTTP access token service is required")
	}

	handler := &Handler{
		app: application,
		profiles: make(
			map[kernel.ProfileCode]http.Handler,
		),
	}
	for _, profile := range application.Definition().Profiles {
		runtime, exists := application.ProfileRuntime(profile.Code)
		if !exists {
			return nil, errors.New(
				"profile runtime " + strconv.Quote(string(profile.Code)) +
					" is unavailable; app must be booted",
			)
		}
		compiled, err := CompileProfile(
			context.Background(),
			runtime,
		)
		if err != nil {
			return nil, err
		}
		handler.profiles[profile.Code] = compiled
	}

	root := chi.NewRouter()
	root.Use(chimiddleware.RequestID)
	root.Use(chimiddleware.ClientIPFromRemoteAddr)
	root.Use(httplog.RequestLogger(
		accessLogger(application.Logger()),
		accessLogOptions(),
	))
	root.Use(requestLogMetadata)
	for _, middleware := range config.platformMiddleware {
		root.Use(middleware)
	}
	root.Use(optionalAuthentication(config.accessTokens))
	root.Use(requestActorLogAttributes)
	login, err := newLoginHandler(application, config.accessTokens)
	if err != nil {
		return nil, err
	}
	root.Method(http.MethodPost, "/api/auth/login", login)
	adminSession, err := newAdminSessionHandler(application)
	if err != nil {
		return nil, err
	}
	root.Method(http.MethodGet, "/api/admin/session", adminSession)

	platform := chi.NewRouter()
	platform.HandleFunc("/files/*", handler.serveFile)
	platform.Get("/runtime", handler.serveRuntime)
	platform.NotFound(http.NotFound)
	root.Mount("/_cms", platform)

	root.NotFound(http.HandlerFunc(handler.dispatchProfile))
	handler.root = root
	return handler, nil
}

func newAdminSessionHandler(
	application *app.App,
) (http.Handler, error) {
	for _, profile := range application.Definition().Profiles {
		runtime, exists := application.ProfileRuntime(profile.Code)
		if !exists {
			return nil, errors.New(
				"admin profile runtime is unavailable",
			)
		}
		moduleRuntime, exists := runtime.Registry().Module(admin.ModuleCode)
		if !exists {
			return nil, errors.New("admin module runtime is unavailable")
		}
		adminRuntime, valid := moduleRuntime.(*admin.Runtime)
		if !valid {
			return nil, errors.New("admin module runtime has invalid type")
		}
		return adminRuntime.SessionHandler()
	}
	return nil, errors.New("admin profile is unavailable")
}

func (h *Handler) ServeHTTP(
	response http.ResponseWriter,
	request *http.Request,
) {
	if h == nil || h.root == nil {
		http.Error(
			response,
			"HTTP handler is unavailable",
			http.StatusInternalServerError,
		)
		return
	}
	h.root.ServeHTTP(response, request)
}

func (h *Handler) serveRuntime(
	response http.ResponseWriter,
	request *http.Request,
) {
	actor, exists := httptransport.ActorFromContext(request.Context())
	if !exists {
		http.Error(
			response,
			"request actor is unavailable",
			http.StatusInternalServerError,
		)
		return
	}
	runtime, err := h.app.RuntimeByDomain(
		request.Context(),
		actor,
		request.Host,
	)
	if err != nil {
		switch {
		case errors.Is(err, site.ErrNotFound):
			http.Error(
				response,
				"site runtime not found",
				http.StatusNotFound,
			)
		case errors.Is(err, security.ErrForbidden),
			errors.Is(err, security.ErrUnauthenticated):
			http.Error(
				response,
				"site runtime forbidden",
				http.StatusForbidden,
			)
		default:
			http.Error(
				response,
				"site runtime failed",
				http.StatusInternalServerError,
			)
		}
		return
	}

	item := runtime.Site()
	response.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(response).Encode(runtimeResponse{
		SiteID:      item.ID,
		Domain:      item.Domain,
		Locale:      item.Locale,
		ProfileCode: runtime.Profile().Profile().Code,
		Settings:    item.Settings,
	}); err != nil {
		http.Error(response, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) dispatchProfile(
	response http.ResponseWriter,
	request *http.Request,
) {
	actor, exists := httptransport.ActorFromContext(request.Context())
	if !exists {
		http.Error(
			response,
			"request actor is unavailable",
			http.StatusInternalServerError,
		)
		return
	}
	runtime, err := h.app.RuntimeByDomain(
		request.Context(),
		actor,
		request.Host,
	)
	if err != nil {
		switch {
		case errors.Is(err, site.ErrNotFound):
			http.NotFound(response, request)
		case errors.Is(err, security.ErrForbidden),
			errors.Is(err, security.ErrUnauthenticated):
			http.Error(response, "site forbidden", http.StatusForbidden)
		default:
			http.Error(
				response,
				"site runtime failed",
				http.StatusInternalServerError,
			)
		}
		return
	}

	profileCode := runtime.Profile().Profile().Code
	profileHandler, exists := h.profiles[profileCode]
	if !exists {
		http.Error(
			response,
			"profile HTTP handler is unavailable",
			http.StatusInternalServerError,
		)
		return
	}
	profileHandler.ServeHTTP(
		response,
		request.WithContext(
			httptransport.WithSiteRuntime(
				request.Context(),
				runtime,
			),
		),
	)
}

func (h *Handler) serveFile(
	response http.ResponseWriter,
	request *http.Request,
) {
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		response.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rawID := strings.TrimPrefix(request.URL.Path, "/_cms/files/")
	if rawID == "" || strings.Contains(rawID, "/") {
		http.NotFound(response, request)
		return
	}
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		http.NotFound(response, request)
		return
	}

	var expiresAt time.Time
	if rawExpires := request.URL.Query().Get("expires"); rawExpires != "" {
		expires, err := strconv.ParseInt(rawExpires, 10, 64)
		if err != nil {
			http.NotFound(response, request)
			return
		}
		expiresAt = time.Unix(expires, 0)
	}

	opened, err := h.app.OpenFileDelivery(
		request.Context(),
		corefile.ID(id),
		corefile.DeliveryAuthorization{
			ExpiresAt: expiresAt,
			Signature: request.URL.Query().Get("signature"),
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, corefile.ErrNotFound),
			errors.Is(err, corefile.ErrUnauthorized):
			http.NotFound(response, request)
		default:
			http.Error(
				response,
				"file delivery failed",
				http.StatusInternalServerError,
			)
		}
		return
	}
	defer func() { _ = opened.Body.Close() }()

	response.Header().Set("Content-Type", opened.File.MIMEType)
	response.Header().Set(
		"Content-Length",
		strconv.FormatInt(opened.File.Size, 10),
	)
	response.Header().Set(
		"Content-Disposition",
		mime.FormatMediaType("inline", map[string]string{
			"filename": opened.File.Name,
		}),
	)
	response.Header().Set("X-Content-Type-Options", "nosniff")
	if request.Method == http.MethodHead {
		response.WriteHeader(http.StatusOK)
		return
	}
	_, _ = io.Copy(response, opened.Body)
}

var _ http.Handler = (*Handler)(nil)
