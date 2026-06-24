package controllers

import (
	"net/http"
	"testing"

	"github.com/vernal96/go-cms/core"
)

func TestPageControllerRoute(t *testing.T) {
	controller := NewPageController()
	if controller.reader == nil {
		t.Fatal("page controller reader is nil")
	}
	if controller.renderer == nil {
		t.Fatal("page controller renderer is nil")
	}

	routes := controller.Routes()
	if len(routes) != 1 {
		t.Fatalf("unexpected route count: %d", len(routes))
	}
	if routes[0].Method != core.RouteMethodGet ||
		string(routes[0].Method) != http.MethodGet {
		t.Fatalf("unexpected route method: %q", routes[0].Method)
	}
	if routes[0].Path != PageRoutePath {
		t.Fatalf("unexpected route path: %q", routes[0].Path)
	}
	if routes[0].Handler == nil {
		t.Fatal("page route handler is nil")
	}
}
