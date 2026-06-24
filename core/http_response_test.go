package core

import (
	"net/http"
	"testing"
)

func TestHTMLResponse(t *testing.T) {
	response := HTMLResponse("<h1>Hello</h1>")

	if response.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", response.StatusCode)
	}
	if response.ContentType != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type: %q", response.ContentType)
	}
	if string(response.Body) != "<h1>Hello</h1>" {
		t.Fatalf("unexpected body: %q", response.Body)
	}
}
