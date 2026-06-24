package core

import (
	"context"
	"strings"
	"testing"
)

type testResourceTemplateRenderer struct {
	resourceType ResourceType
	template     ResourceTemplateCode
}

func (r testResourceTemplateRenderer) ResourceType() ResourceType {
	return r.resourceType
}

func (r testResourceTemplateRenderer) ResourceTemplate() ResourceTemplateCode {
	return r.template
}

func (r testResourceTemplateRenderer) Render(
	ctx context.Context,
	runtime *SiteRuntime,
	data ResourceData,
) (string, error) {
	return "", nil
}

func TestRuntimeResourceTemplateRendererRegistryRegister(t *testing.T) {
	registry := newTestResourceTemplateRendererRegistry(t)
	renderer := testResourceTemplateRenderer{
		resourceType: "page",
		template:     "default",
	}

	if err := registry.ResourceTemplateRenderers().Register(renderer); err != nil {
		t.Fatal(err)
	}

	registered, exists := registry.ResourceTemplateRenderers().Get("page", "default")
	if !exists {
		t.Fatal("registered resource template renderer not found")
	}
	if registered.ResourceTemplate() != "default" {
		t.Fatalf("unexpected renderer template: %q", registered.ResourceTemplate())
	}
}

func TestRuntimeResourceTemplateRendererRegistryValidatesRenderer(t *testing.T) {
	t.Run("nil renderer", func(t *testing.T) {
		err := NewRuntimeRegistry().ResourceTemplateRenderers().Register(nil)
		if err == nil {
			t.Fatal("expected nil renderer error")
		}
	})

	t.Run("empty resource type", func(t *testing.T) {
		err := NewRuntimeRegistry().ResourceTemplateRenderers().Register(
			testResourceTemplateRenderer{template: "default"},
		)
		if err == nil {
			t.Fatal("expected empty resource type error")
		}
	})

	t.Run("empty resource template", func(t *testing.T) {
		err := NewRuntimeRegistry().ResourceTemplateRenderers().Register(
			testResourceTemplateRenderer{resourceType: "page"},
		)
		if err == nil {
			t.Fatal("expected empty resource template error")
		}
	})

	t.Run("unknown resource type", func(t *testing.T) {
		err := NewRuntimeRegistry().ResourceTemplateRenderers().Register(
			testResourceTemplateRenderer{
				resourceType: "page",
				template:     "default",
			},
		)
		if err == nil || !strings.Contains(err.Error(), "resource type") {
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

		err := registry.ResourceTemplateRenderers().Register(
			testResourceTemplateRenderer{
				resourceType: "page",
				template:     "default",
			},
		)
		if err == nil || !strings.Contains(err.Error(), "resource template") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestRuntimeResourceTemplateRendererRegistryRejectsDuplicate(t *testing.T) {
	registry := newTestResourceTemplateRendererRegistry(t)
	renderer := testResourceTemplateRenderer{
		resourceType: "page",
		template:     "default",
	}

	if err := registry.ResourceTemplateRenderers().Register(renderer); err != nil {
		t.Fatal(err)
	}

	err := registry.ResourceTemplateRenderers().Register(renderer)
	if err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeResourceTemplateRendererRegistryAllForResourceType(t *testing.T) {
	registry := NewRuntimeRegistry()
	if err := registry.ResourceTypes().Register(testResourceType{
		code: "page",
		name: "Page",
	}); err != nil {
		t.Fatal(err)
	}

	for _, template := range []ResourceTemplateCode{"zeta", "alpha"} {
		if err := registry.ResourceTemplates().Register(testResourceTemplate{
			code:         template,
			name:         string(template),
			resourceType: "page",
		}); err != nil {
			t.Fatal(err)
		}
		if err := registry.ResourceTemplateRenderers().Register(
			testResourceTemplateRenderer{
				resourceType: "page",
				template:     template,
			},
		); err != nil {
			t.Fatal(err)
		}
	}

	renderers := registry.ResourceTemplateRenderers().AllForResourceType("page")
	if len(renderers) != 2 {
		t.Fatalf("unexpected renderer count: %d", len(renderers))
	}
	if renderers[0].ResourceTemplate() != "alpha" ||
		renderers[1].ResourceTemplate() != "zeta" {
		t.Fatalf("renderers are not sorted: %#v", renderers)
	}
}

func newTestResourceTemplateRendererRegistry(t *testing.T) *RuntimeRegistry {
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
