package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/app"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
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
	if request.URL.Path != "/_cms/runtime" {
		http.NotFound(response, request)
		return
	}

	if request.Method != http.MethodGet {
		response.Header().Set("Allow", http.MethodGet)
		http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	runtime, exists := h.app.RuntimeByDomain(request.Host)
	if !exists {
		http.Error(response, "site runtime not found", http.StatusNotFound)
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

var _ http.Handler = (*Handler)(nil)
