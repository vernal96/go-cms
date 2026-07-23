package httpserver

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/app"
	corefile "github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/security"
)

type Handler struct {
	app *app.App
}

type runtimeResponse struct {
	SiteID      site.ID            `json:"site_id"`
	Domain      string             `json:"domain"`
	Locale      string             `json:"locale"`
	ProfileCode kernel.ProfileCode `json:"profile_code"`
	Settings    map[string]any     `json:"settings"`
}

func NewHandler(app *app.App) (*Handler, error) {
	if app == nil {
		return nil, errors.New("app is nil")
	}

	return &Handler{app: app}, nil
}

func (h *Handler) ServeHTTP(
	response http.ResponseWriter,
	request *http.Request,
) {
	if strings.HasPrefix(request.URL.Path, "/_cms/files/") {
		h.serveFile(response, request)
		return
	}

	if request.URL.Path != "/_cms/runtime" {
		http.NotFound(response, request)
		return
	}

	if request.Method != http.MethodGet {
		response.Header().Set("Allow", http.MethodGet)
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	runtime, err := h.app.RuntimeByDomain(
		request.Context(),
		security.Guest(),
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
