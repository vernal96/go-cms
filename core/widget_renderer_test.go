package core

import (
	"context"
	"errors"
	"testing"
)

func TestNewWidgetRenderer(t *testing.T) {
	if NewWidgetRenderer() == nil {
		t.Fatal("widget renderer is nil")
	}
}

func TestWidgetRendererValidatesInput(t *testing.T) {
	renderer := NewWidgetRenderer()

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := renderer.RenderArea(ctx, nil, ResourceData{}, "main")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("nil runtime", func(t *testing.T) {
		_, err := renderer.RenderArea(
			context.Background(),
			nil,
			ResourceData{},
			"main",
		)
		if err == nil || err.Error() != "site runtime is nil" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestWidgetRendererRejectsUnknownWidget(t *testing.T) {
	runtime := newResourceReaderRuntimeWithRegistry(
		t,
		&readerResourceRepository{},
		&readerResourceFieldValueRepository{},
		NewRuntimeRegistry(),
	)

	_, err := NewWidgetRenderer().RenderArea(
		context.Background(),
		runtime,
		ResourceData{
			Widgets: []WidgetInstance{
				{
					Widget: "unknown.widget",
					Area:   "main",
				},
			},
		},
		"main",
	)
	if err == nil || err.Error() != `widget "unknown.widget" is not registered` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWidgetRendererRejectsUnknownWidgetTemplate(t *testing.T) {
	registry := NewRuntimeRegistry()
	if err := registry.ForModule("test").Widgets().Register(testHTMLWidget{}); err != nil {
		t.Fatal(err)
	}
	runtime := newResourceReaderRuntimeWithRegistry(
		t,
		&readerResourceRepository{},
		&readerResourceFieldValueRepository{},
		registry,
	)

	_, err := NewWidgetRenderer().RenderArea(
		context.Background(),
		runtime,
		ResourceData{
			Widgets: []WidgetInstance{
				{
					Widget:   "test.html",
					Template: "custom",
					Area:     "main",
				},
			},
		},
		"main",
	)
	if err == nil ||
		err.Error() != `widget template "custom" for widget "test.html" is not registered` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWidgetRendererFiltersAreaAndRendersWidgets(t *testing.T) {
	registry := NewRuntimeRegistry()
	moduleRegistry := registry.ForModule("test")
	if err := moduleRegistry.Widgets().Register(testHTMLWidget{}); err != nil {
		t.Fatal(err)
	}
	runtime := newResourceReaderRuntimeWithRegistry(
		t,
		&readerResourceRepository{},
		&readerResourceFieldValueRepository{},
		registry,
	)

	html, err := NewWidgetRenderer().RenderArea(
		context.Background(),
		runtime,
		ResourceData{
			Widgets: []WidgetInstance{
				{
					ID:     1,
					Widget: "unknown.sidebar",
					Area:   "sidebar",
				},
				{
					ID:       2,
					Widget:   "test.html",
					Template: WidgetTemplateDefault,
					Area:     "main",
					Params: WidgetParams{
						"text": "First",
					},
				},
				{
					ID:     3,
					Widget: "test.html",
					Area:   "main",
					Params: WidgetParams{
						"text": "Second",
					},
				},
			},
		},
		"main",
	)
	if err != nil {
		t.Fatal(err)
	}
	if html != "<p>First</p><p>Second</p>" {
		t.Fatalf("unexpected rendered widgets: %q", html)
	}
}

type testHTMLWidget struct{}

func (testHTMLWidget) Code() WidgetCode {
	return "html"
}

func (testHTMLWidget) Name() string {
	return "HTML"
}

func (testHTMLWidget) Params() []WidgetParamDefinition {
	return nil
}

func (testHTMLWidget) Render(
	ctx context.Context,
	params WidgetParams,
) (WidgetResult, error) {
	return WidgetResult{
		HTML: "<p>" + params["text"].(string) + "</p>",
	}, nil
}
