package core

import (
	"context"
	"errors"
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

	t.Run("resource template renderer", func(t *testing.T) {
		registry := newResourceRendererRegistry(t)
		runtime := newResourceRendererRuntime(t, registry)

		_, err := renderer.Render(context.Background(), runtime, data)
		if err == nil ||
			err.Error() != `resource template renderer for resource type "page" and template "default" is not registered` {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestResourceRendererDispatchesToRegisteredRenderer(t *testing.T) {
	registry := newResourceRendererRegistry(t)
	templateRenderer := &recordingResourceTemplateRenderer{
		output: "<html>rendered</html>",
	}
	if err := registry.ResourceTemplateRenderers().Register(templateRenderer); err != nil {
		t.Fatal(err)
	}
	runtime := newResourceRendererRuntime(t, registry)
	data := ResourceData{
		Resource: Resource{
			ID:       1,
			Type:     "page",
			Template: "default",
			Title:    "Home",
		},
	}

	output, err := NewResourceRenderer().Render(context.Background(), runtime, data)
	if err != nil {
		t.Fatal(err)
	}
	if output != templateRenderer.output {
		t.Fatalf("unexpected output: %q", output)
	}
	if !templateRenderer.called {
		t.Fatal("registered renderer was not called")
	}
	if templateRenderer.runtime != runtime {
		t.Fatal("renderer received a different runtime")
	}
	if templateRenderer.data.Resource.ID != data.Resource.ID {
		t.Fatalf("renderer received unexpected data: %#v", templateRenderer.data)
	}
}

func newResourceRendererRegistry(t *testing.T) *RuntimeRegistry {
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

type recordingResourceTemplateRenderer struct {
	output  string
	called  bool
	runtime *SiteRuntime
	data    ResourceData
}

func (r *recordingResourceTemplateRenderer) ResourceType() ResourceType {
	return "page"
}

func (r *recordingResourceTemplateRenderer) ResourceTemplate() ResourceTemplateCode {
	return "default"
}

func (r *recordingResourceTemplateRenderer) Render(
	ctx context.Context,
	runtime *SiteRuntime,
	data ResourceData,
) (string, error) {
	r.called = true
	r.runtime = runtime
	r.data = data

	return r.output, nil
}
