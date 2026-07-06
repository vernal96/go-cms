package http

import (
	"fmt"
	"net/http"

	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

type Handler struct {
	runtimeProvider *site.RuntimeProvider
}

func NewHandler(runtimeResolver *site.RuntimeProvider) (*Handler, error) {
	if runtimeResolver == nil {
		return nil, fmt.Errorf("runtime resolver is nil")
	}

	return &Handler{
		runtimeProvider: runtimeResolver,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	runtime, exists, err := h.runtimeProvider.ResolveByDomain(r.Context(), r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "site not found", http.StatusNotFound)
		return
	}

	_, _ = fmt.Fprintf(w, "runtime created for profile: %s", runtime.Profile().Code())
}

var _ http.Handler = (*Handler)(nil)
