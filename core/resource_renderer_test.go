package core

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestNewResourceRenderer(t *testing.T) {
	if NewResourceRenderer() == nil {
		t.Fatal("resource renderer is nil")
	}
}

func TestResourceRendererValidatesInput(t *testing.T) {
	renderer := NewResourceRenderer()

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := renderer.Render(ctx, nil, ResourceData{})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("nil runtime", func(t *testing.T) {
		_, err := renderer.Render(context.Background(), nil, ResourceData{})
		if err == nil || err.Error() != "site runtime is nil" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("resource id", func(t *testing.T) {
		runtime := newResourceRendererRuntime(t, NewRuntimeRegistry())

		_, err := renderer.Render(context.Background(), runtime, ResourceData{})
		if err == nil || err.Error() != "resource id must be positive" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestResourceRendererValidatesRegistry(t *testing.T) {
	renderer := NewResourceRenderer()
	data := ResourceData{
		Resource: Resource{
			ID:       1,
			Type:     "page",
			Template: "default",
		},
	}

	t.Run("resource type", func(t *testing.T) {
		runtime := newResourceRendererRuntime(t, NewRuntimeRegistry())

		_, err := renderer.Render(context.Background(), runtime, data)
		if err == nil || err.Error() != `resource type "page" is not registered` {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("resource template", func(t *testing.T) {
		registry := NewRuntimeRegistry()
		if err := registry.ResourceTypes().Register(readerResourceType{}); err != nil {
			t.Fatal(err)
		}
		runtime := newResourceRendererRuntime(t, registry)

		_, err := renderer.Render(context.Background(), runtime, data)
		if err == nil ||
			err.Error() != `resource template "default" for resource type "page" is not registered` {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestResourceRendererRendersDefaultPage(t *testing.T) {
	runtime := newResourceRendererRuntime(t, newDefaultPageRegistry(t))
	data := ResourceData{
		Resource: Resource{
			ID:       1,
			Type:     "page",
			Template: "default",
			Title:    "Home",
		},
		Values: []ResourceFieldValue{
			{
				ResourceID: 1,
				Field:      "content",
				Value:      "Hello world",
			},
		},
	}

	html, err := NewResourceRenderer().Render(context.Background(), runtime, data)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(html, "<title>Home</title>") {
		t.Fatalf("rendered HTML does not contain title: %s", html)
	}
	if !strings.Contains(html, "<h1>Home</h1>") {
		t.Fatalf("rendered HTML does not contain heading: %s", html)
	}
	if !strings.Contains(html, "<div>Hello world</div>") {
		t.Fatalf("rendered HTML does not contain content: %s", html)
	}
}

func TestResourceRendererRejectsUnsupportedRegisteredTemplate(t *testing.T) {
	registry := NewRuntimeRegistry()
	if err := registry.ResourceTypes().Register(rendererCollectionType{}); err != nil {
		t.Fatal(err)
	}
	if err := registry.ResourceTemplates().Register(rendererCollectionTemplate{}); err != nil {
		t.Fatal(err)
	}
	runtime := newResourceRendererRuntime(t, registry)

	_, err := NewResourceRenderer().Render(context.Background(), runtime, ResourceData{
		Resource: Resource{
			ID:       1,
			Type:     "collection",
			Template: "list",
		},
	})
	if err == nil ||
		err.Error() != `resource renderer does not support resource type "collection" and template "list"` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResourceRendererEscapesHTML(t *testing.T) {
	runtime := newResourceRendererRuntime(t, newDefaultPageRegistry(t))
	data := ResourceData{
		Resource: Resource{
			ID:       1,
			Type:     "page",
			Template: "default",
			Title:    "<b>Home</b>",
		},
		Values: []ResourceFieldValue{
			{
				ResourceID: 1,
				Field:      "content",
				Value:      "<script>alert(1)</script>",
			},
		},
	}

	html, err := NewResourceRenderer().Render(context.Background(), runtime, data)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(html, "<script>") {
		t.Fatalf("rendered HTML contains a raw script tag: %s", html)
	}
	if !strings.Contains(html, "&lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Fatalf("rendered HTML does not contain escaped content: %s", html)
	}
	if strings.Contains(html, "<b>Home</b>") {
		t.Fatalf("rendered HTML contains a raw title tag: %s", html)
	}
}

func newDefaultPageRegistry(t *testing.T) *RuntimeRegistry {
	t.Helper()

	registry := NewRuntimeRegistry()
	if err := registry.ResourceTypes().Register(readerResourceType{}); err != nil {
		t.Fatal(err)
	}
	if err := registry.ResourceTemplates().Register(readerResourceTemplate{}); err != nil {
		t.Fatal(err)
	}

	return registry
}

func newResourceRendererRuntime(t *testing.T, registry Registry) *SiteRuntime {
	t.Helper()

	return newResourceReaderRuntimeWithRegistry(
		t,
		&readerResourceRepository{},
		&readerResourceFieldValueRepository{},
		registry,
	)
}

type rendererCollectionType struct{}

func (rendererCollectionType) Code() ResourceType {
	return "collection"
}

func (rendererCollectionType) Name() string {
	return "Collection"
}

type rendererCollectionTemplate struct{}

func (rendererCollectionTemplate) Code() ResourceTemplateCode {
	return "list"
}

func (rendererCollectionTemplate) Name() string {
	return "Collection list"
}

func (rendererCollectionTemplate) ResourceType() ResourceType {
	return "collection"
}
