package controllers

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/vernal96/go-cms/core"
	modulefields "github.com/vernal96/go-cms/core/modules/core/fields"
	"github.com/vernal96/go-cms/core/modules/core/resources"
)

func TestResourceControllerRoute(t *testing.T) {
	controller := NewResourceController()
	if controller.reader == nil {
		t.Fatal("resource controller reader is nil")
	}

	routes := controller.Routes()

	if len(routes) != 1 {
		t.Fatalf("unexpected route count: %d", len(routes))
	}
	if routes[0].Method != core.RouteMethodGet {
		t.Fatalf("unexpected route method: %q", routes[0].Method)
	}
	if string(routes[0].Method) != http.MethodGet {
		t.Fatalf("route method must be GET: %q", routes[0].Method)
	}
	if routes[0].Path != ResourceRoutePath {
		t.Fatalf("unexpected route path: %q", routes[0].Path)
	}
	if routes[0].Handler == nil {
		t.Fatal("resource route handler is nil")
	}
}

func TestResourceFieldDefinitionsResponse(t *testing.T) {
	response := resourceFieldDefinitionsResponse([]core.ResourceFieldDefinition{
		resources.NewPageContentField(),
	})

	expected := []resourceFieldDefinitionResponse{
		{
			Code:             resources.PageContentFieldCode,
			Name:             "Content",
			FieldType:        string(modulefields.TextFieldTypeCode),
			ResourceType:     resources.PageResourceTypeCode,
			ResourceTemplate: resources.PageDefaultTemplateCode,
			Required:         false,
		},
	}

	if !reflect.DeepEqual(response, expected) {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestResourceFieldValuesResponse(t *testing.T) {
	value := map[string]any{
		"blocks": []any{"Hello"},
	}

	response := resourceFieldValuesResponse([]core.ResourceFieldValue{
		{
			ResourceID: 10,
			Field:      resources.PageContentFieldCode,
			Value:      value,
		},
	})

	expected := []resourceFieldValueResponse{
		{
			Field: resources.PageContentFieldCode,
			Value: value,
		},
	}

	if !reflect.DeepEqual(response, expected) {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestResourceFieldResponsesAreEmptySlices(t *testing.T) {
	if response := resourceFieldDefinitionsResponse(nil); response == nil || len(response) != 0 {
		t.Fatalf("expected empty field definitions response, got %#v", response)
	}

	if response := resourceFieldValuesResponse(nil); response == nil || len(response) != 0 {
		t.Fatalf("expected empty field values response, got %#v", response)
	}
}
