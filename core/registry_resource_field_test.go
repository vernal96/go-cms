package core

import (
	"strings"
	"testing"

	corefields "github.com/vernal96/go-cms/core/fields"
)

type testResourceField struct {
	code             ResourceFieldCode
	name             string
	field            corefields.FieldType
	resourceType     ResourceType
	resourceTemplate ResourceTemplateCode
	required         bool
}

func (f testResourceField) Code() ResourceFieldCode {
	return f.code
}

func (f testResourceField) Name() string {
	return f.name
}

func (f testResourceField) Field() corefields.FieldType {
	return f.field
}

func (f testResourceField) ResourceType() ResourceType {
	return f.resourceType
}

func (f testResourceField) ResourceTemplate() ResourceTemplateCode {
	return f.resourceTemplate
}

func (f testResourceField) Required() bool {
	return f.required
}

func TestRuntimeResourceFieldRegistryRegister(t *testing.T) {
	registry := newTestPageRegistry(t)
	field := validTestResourceField()

	if err := registry.ResourceFields().Register(field); err != nil {
		t.Fatal(err)
	}

	registered, exists := registry.ResourceFields().Get("page", "default", "content")
	if !exists {
		t.Fatal("registered resource field not found")
	}
	if registered.Name() != "Content" {
		t.Fatalf("unexpected resource field name: %q", registered.Name())
	}
	if registered.Required() {
		t.Fatal("content resource field must not be required")
	}

	all := registry.ResourceFields().AllForTemplate("page", "default")
	if len(all) != 1 || all[0].Code() != "content" {
		t.Fatalf("unexpected registered resource fields: %#v", all)
	}
}

func TestRuntimeResourceFieldRegistryValidatesField(t *testing.T) {
	t.Run("nil field", func(t *testing.T) {
		err := NewRuntimeRegistry().ResourceFields().Register(nil)
		if err == nil {
			t.Fatal("expected nil resource field error")
		}
	})

	t.Run("empty code", func(t *testing.T) {
		registry := newTestPageRegistry(t)
		field := validTestResourceField()
		field.code = ""

		if err := registry.ResourceFields().Register(field); err == nil {
			t.Fatal("expected empty resource field code error")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		registry := newTestPageRegistry(t)
		field := validTestResourceField()
		field.name = ""

		if err := registry.ResourceFields().Register(field); err == nil {
			t.Fatal("expected empty resource field name error")
		}
	})

	t.Run("nil field type", func(t *testing.T) {
		registry := newTestPageRegistry(t)
		field := validTestResourceField()
		field.field = nil

		if err := registry.ResourceFields().Register(field); err == nil {
			t.Fatal("expected nil field type error")
		}
	})

	t.Run("empty resource type", func(t *testing.T) {
		field := validTestResourceField()
		field.resourceType = ""

		if err := NewRuntimeRegistry().ResourceFields().Register(field); err == nil {
			t.Fatal("expected empty resource type error")
		}
	})

	t.Run("empty resource template", func(t *testing.T) {
		field := validTestResourceField()
		field.resourceTemplate = ""

		if err := NewRuntimeRegistry().ResourceFields().Register(field); err == nil {
			t.Fatal("expected empty resource template error")
		}
	})

	t.Run("unknown resource type", func(t *testing.T) {
		field := validTestResourceField()

		err := NewRuntimeRegistry().ResourceFields().Register(field)
		if err == nil {
			t.Fatal("expected unknown resource type error")
		}
		if !strings.Contains(err.Error(), "resource type") ||
			!strings.Contains(err.Error(), "not registered") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("unknown resource template", func(t *testing.T) {
		registry := NewRuntimeRegistry()
		if err := registry.ResourceTypes().Register(testResourceType{
			code: "page",
			name: "Page",
		}); err != nil {
			t.Fatal(err)
		}

		err := registry.ResourceFields().Register(validTestResourceField())
		if err == nil {
			t.Fatal("expected unknown resource template error")
		}
		if !strings.Contains(err.Error(), "resource template") ||
			!strings.Contains(err.Error(), "not registered") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestRuntimeResourceFieldRegistryRejectsDuplicate(t *testing.T) {
	registry := newTestPageRegistry(t)
	field := validTestResourceField()

	if err := registry.ResourceFields().Register(field); err != nil {
		t.Fatal(err)
	}

	err := registry.ResourceFields().Register(field)
	if err == nil {
		t.Fatal("expected duplicate resource field error")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func newTestPageRegistry(t *testing.T) *RuntimeRegistry {
	t.Helper()

	registry := NewRuntimeRegistry()
	if err := registry.ResourceTypes().Register(testResourceType{
		code: "page",
		name: "Page",
	}); err != nil {
		t.Fatal(err)
	}
	if err := registry.ResourceTemplates().Register(testResourceTemplate{
		code:         "default",
		name:         "Default page",
		resourceType: "page",
	}); err != nil {
		t.Fatal(err)
	}

	return registry
}

func validTestResourceField() testResourceField {
	return testResourceField{
		code:             "content",
		name:             "Content",
		field:            corefields.NewInput(),
		resourceType:     "page",
		resourceTemplate: "default",
	}
}
