package controllers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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
	if controller.writer == nil {
		t.Fatal("resource controller writer is nil")
	}

	routes := controller.Routes()

	if len(routes) != 2 {
		t.Fatalf("unexpected route count: %d", len(routes))
	}

	routesByPath := make(map[string]core.Route, len(routes))
	for _, route := range routes {
		routesByPath[route.Path] = route
	}

	resourceRoute, exists := routesByPath[ResourceRoutePath]
	if !exists {
		t.Fatal("resource route is not registered")
	}
	if resourceRoute.Method != core.RouteMethodGet ||
		string(resourceRoute.Method) != http.MethodGet {
		t.Fatalf("unexpected resource route method: %q", resourceRoute.Method)
	}
	if resourceRoute.Handler == nil {
		t.Fatal("resource route handler is nil")
	}

	fieldValueRoute, exists := routesByPath[ResourceFieldValueRoutePath]
	if !exists {
		t.Fatal("resource field value route is not registered")
	}
	if fieldValueRoute.Method != core.RouteMethodPost ||
		string(fieldValueRoute.Method) != http.MethodPost {
		t.Fatalf(
			"unexpected resource field value route method: %q",
			fieldValueRoute.Method,
		)
	}
	if fieldValueRoute.Handler == nil {
		t.Fatal("resource field value route handler is nil")
	}
}

func TestDecodeSaveResourceFieldValueRequest(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		ResourceFieldValueRoutePath,
		bytes.NewBufferString(`{
			"path": "/",
			"field": "content",
			"value": "Hello world"
		}`),
	)

	input, err := decodeSaveResourceFieldValueRequest(request)
	if err != nil {
		t.Fatal(err)
	}

	expected := saveResourceFieldValueRequest{
		Path:  "/",
		Field: resources.PageContentFieldCode,
		Value: "Hello world",
	}
	if !reflect.DeepEqual(input, expected) {
		t.Fatalf("unexpected request: %#v", input)
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

func TestResourceFieldValueResponseFromValue(t *testing.T) {
	value := map[string]any{
		"text": "Hello",
	}

	response := resourceFieldValueResponseFromValue(core.ResourceFieldValue{
		ResourceID: 10,
		Field:      resources.PageContentFieldCode,
		Value:      value,
	})

	expected := resourceFieldValueResponse{
		Field: resources.PageContentFieldCode,
		Value: value,
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
