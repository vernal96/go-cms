package core

import (
	"strings"
	"testing"
)

type testResourceTemplate struct {
	code         ResourceTemplateCode
	name         string
	resourceType ResourceType
}

func (t testResourceTemplate) Code() ResourceTemplateCode {
	return t.code
}

func (t testResourceTemplate) Name() string {
	return t.name
}

func (t testResourceTemplate) ResourceType() ResourceType {
	return t.resourceType
}

func TestRuntimeResourceTemplateRegistryRegister(t *testing.T) {
	registry := NewRuntimeRegistry()
	if err := registry.ResourceTypes().Register(testResourceType{
		code: "page",
		name: "Page",
	}); err != nil {
		t.Fatal(err)
	}

	template := testResourceTemplate{
		code:         "default",
		name:         "Default page",
		resourceType: "page",
	}
	if err := registry.ResourceTemplates().Register(template); err != nil {
		t.Fatal(err)
	}

	registered, exists := registry.ResourceTemplates().Get("page", "default")
	if !exists {
		t.Fatal("registered resource template not found")
	}
	if registered.Name() != "Default page" {
		t.Fatalf("unexpected resource template name: %q", registered.Name())
	}

	all := registry.ResourceTemplates().AllForResourceType("page")
	if len(all) != 1 || all[0].Code() != "default" {
		t.Fatalf("unexpected registered resource templates: %#v", all)
	}
}

func TestRuntimeResourceTemplateRegistryValidatesTemplate(t *testing.T) {
	t.Run("nil template", func(t *testing.T) {
		err := NewRuntimeRegistry().ResourceTemplates().Register(nil)
		if err == nil {
			t.Fatal("expected nil resource template error")
		}
	})

	t.Run("empty resource type", func(t *testing.T) {
		err := NewRuntimeRegistry().ResourceTemplates().Register(testResourceTemplate{
			code: "default",
			name: "Default page",
		})
		if err == nil {
			t.Fatal("expected empty resource type error")
		}
	})

	t.Run("empty code", func(t *testing.T) {
		err := NewRuntimeRegistry().ResourceTemplates().Register(testResourceTemplate{
			name:         "Default page",
			resourceType: "page",
		})
		if err == nil {
			t.Fatal("expected empty resource template code error")
		}
	})

	t.Run("unregistered resource type", func(t *testing.T) {
		err := NewRuntimeRegistry().ResourceTemplates().Register(testResourceTemplate{
			code:         "default",
			name:         "Default page",
			resourceType: "page",
		})
		if err == nil {
			t.Fatal("expected unregistered resource type error")
		}
		if !strings.Contains(err.Error(), "not registered") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestRuntimeResourceTemplateRegistryRejectsDuplicate(t *testing.T) {
	registry := NewRuntimeRegistry()
	if err := registry.ResourceTypes().Register(testResourceType{
		code: "page",
		name: "Page",
	}); err != nil {
		t.Fatal(err)
	}

	template := testResourceTemplate{
		code:         "default",
		name:         "Default page",
		resourceType: "page",
	}
	if err := registry.ResourceTemplates().Register(template); err != nil {
		t.Fatal(err)
	}

	err := registry.ResourceTemplates().Register(template)
	if err == nil {
		t.Fatal("expected duplicate resource template error")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("unexpected error: %v", err)
	}
}
