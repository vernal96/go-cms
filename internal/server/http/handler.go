package http

import (
	"fmt"
	"net/http"

	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

type Handler struct {
	runtimeResolver *site.RuntimeResolver
}

func NewHandler(runtimeResolver *site.RuntimeResolver) (*Handler, error) {
	if runtimeResolver == nil {
		return nil, fmt.Errorf("runtime resolver is nil")
	}

	return &Handler{
		runtimeResolver: runtimeResolver,
	}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	runtime, exists, err := h.runtimeResolver.ResolveByDomain(r.Context(), r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, "site not found", http.StatusNotFound)
		return
	}

	_, _ = fmt.Fprintf(w, "runtime created for profile: %s", runtime.Profile().Code())

	var _ http.Handler = (*Handler)(nil)
}
