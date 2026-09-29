package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRootHandlerHealthAndAPIRouting(t *testing.T) {
	apiCalls := 0
	handler := rootHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCalls++
		if r.URL.Path != "/api/auth/me" {
			t.Errorf("unexpected API path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if response.Code != http.StatusOK || apiCalls != 0 {
		t.Fatalf("health request: status=%d, API calls=%d", response.Code, apiCalls)
	}
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/auth/me", nil))
	if response.Code != http.StatusUnauthorized || apiCalls != 1 {
		t.Fatalf("API request: status=%d, API calls=%d", response.Code, apiCalls)
	}
}
