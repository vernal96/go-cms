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
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/app"
	"github.com/vernal96/go-cms/kernel/modules/admin"
	"github.com/vernal96/go-cms/kernel/modules/core"
	corefile "github.com/vernal96/go-cms/kernel/modules/core/file"
	coremanagement "github.com/vernal96/go-cms/kernel/modules/core/management"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	mailmodule "github.com/vernal96/go-cms/kernel/modules/mail"
	"github.com/vernal96/go-cms/kernel/security"
	httptransport "github.com/vernal96/go-cms/kernel/transport/http"
)

type Handler struct {
	app          *app.App
	root         http.Handler
	siteHandlers atomic.Pointer[siteHandlerSnapshot]
}

type siteHandlerSnapshot struct {
	byDomain  map[string]compiledSiteRuntime
	byRuntime map[*site.Runtime]http.Handler
}

type compiledSiteRuntime struct {
	runtime *site.Runtime
	handler http.Handler
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

	handler := &Handler{app: application}
	handler.siteHandlers.Store(&siteHandlerSnapshot{
		byDomain:  make(map[string]compiledSiteRuntime),
		byRuntime: make(map[*site.Runtime]http.Handler),
	})
	if application.Sites() == nil {
		return nil, errors.New("site runtime catalog is unavailable; app must be booted")
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
	login, err := newLoginHandler(application.Users(), config.accessTokens)
	if err != nil {
		return nil, err
	}
	root.Method(http.MethodPost, "/api/auth/login", login)
	adminHandler, err := newAdminHandler(application)
	if err != nil {
		return nil, err
	}
	cmsHandler, err := newCMSHandler(application)
	if err != nil {
		return nil, err
	}
	api := chi.NewRouter()
	api.Mount("/admin", adminHandler)
	api.Mount("/", cmsHandler)
	root.Mount("/api", api)

	platform := chi.NewRouter()
	platform.HandleFunc("/files/*", handler.serveFile)
	platform.Get("/runtime", handler.serveRuntime)
	platform.NotFound(http.NotFound)
	root.Mount("/_cms", platform)

	root.NotFound(http.HandlerFunc(handler.dispatchProfile))
	handler.root = root
	if err := application.Sites().AddRuntimePreparer(
		context.Background(),
		handler.prepareRuntimes,
	); err != nil {
		return nil, err
	}
	return handler, nil
}

func (h *Handler) prepareRuntimes(
	ctx context.Context,
	plan site.RuntimePlan,
) (site.RuntimePreparation, error) {
	current := h.siteHandlers.Load()
	if current == nil {
		return site.RuntimePreparation{}, errors.New(
			"site HTTP handler snapshot is unavailable",
		)
	}
	nextHandlers := make(
		map[*site.Runtime]http.Handler,
		len(plan.Next()),
	)
	nextDomains := make(
		map[string]compiledSiteRuntime,
		len(plan.Next()),
	)
	for _, runtime := range plan.Next() {
		compiled, exists := current.byRuntime[runtime]
		if !exists {
			var err error
			compiled, err = CompileSite(ctx, runtime)
			if err != nil {
				return site.RuntimePreparation{}, err
			}
		}
		nextHandlers[runtime] = compiled
		item := runtime.Site()
		nextDomains[item.Domain] = compiledSiteRuntime{
			runtime: runtime,
			handler: compiled,
		}
	}
	next := &siteHandlerSnapshot{
		byDomain:  nextDomains,
		byRuntime: nextHandlers,
	}
	return site.RuntimePreparation{
		Publish: func() { h.siteHandlers.Store(next) },
	}, nil
}

func newAdminHandler(
	application *app.App,
) (http.Handler, error) {
	management, err := application.AdminManagement()
	if err != nil {
		return nil, err
	}
	adminRuntime, err := admin.NewRuntime(
		application.Users(),
		application.Authorization(),
	)
	if err != nil {
		return nil, err
	}
	return adminRuntime.AdminHandler(management)
}

func newCMSHandler(application *app.App) (http.Handler, error) {
	sites, resources, files, err := application.CMSManagement()
	if err != nil {
		return nil, err
	}
	definition := application.Definition()
	coreHandler, err := coremanagement.NewHTTPHandler(coremanagement.HTTPDependencies{
		Sites: sites, Resources: resources, Files: files,
		MaxUploadSize: definition.MaxUploadSize,
		UploadTimeout: definition.UploadTimeout,
	})
	if err != nil {
		return nil, err
	}
	mailManagement, err := application.MailManagement()
	if err != nil {
		return nil, err
	}
	mailHandler, err := mailmodule.NewHTTPHandler(mailManagement)
	if err != nil {
		return nil, err
	}
	router := chi.NewRouter()
	router.Mount("/sites/{siteID}/mail", mailHandler)
	router.Mount("/", coreHandler)
	return router, nil
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
	compiled, exists := h.compiledSiteByDomain(request.Host)
	if !exists {
		http.NotFound(response, request)
		return
	}
	if err := h.app.Sites().CheckReadAccess(
		request.Context(),
		actor,
		compiled.runtime,
	); err != nil {
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
	compiled.handler.ServeHTTP(
		response,
		request.WithContext(
			core.WithSiteRuntime(
				request.Context(),
				compiled.runtime,
			),
		),
	)
}

func (h *Handler) compiledSiteByDomain(
	domain string,
) (compiledSiteRuntime, bool) {
	if h == nil {
		return compiledSiteRuntime{}, false
	}
	domain, err := site.NormalizeDomain(domain)
	if err != nil {
		return compiledSiteRuntime{}, false
	}
	snapshot := h.siteHandlers.Load()
	if snapshot == nil {
		return compiledSiteRuntime{}, false
	}
	compiled, exists := snapshot.byDomain[domain]
	return compiled, exists
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

	opened, err := h.app.Files().OpenDelivery(
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
