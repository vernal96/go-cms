package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/vernal96/go-cms/core"
	corewidgets "github.com/vernal96/go-cms/core/modules/core/widgets"
)

func TestPageDefaultRendererRendersTitleAndContent(t *testing.T) {
	runtime := newPageRendererRuntime(t)
	data := core.ResourceData{
		Resource: core.Resource{
			ID:    1,
			Title: "Home",
		},
		Values: []core.ResourceFieldValue{
			{
				ResourceID: 1,
				Field:      PageContentFieldCode,
				Value:      "Hello world",
			},
		},
	}

	html, err := NewPageDefaultRenderer().Render(
		context.Background(),
		runtime,
		data,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(html, `<html lang="ru">`) {
		t.Fatalf("rendered HTML does not contain locale: %s", html)
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

func TestPageDefaultRendererEscapesHTML(t *testing.T) {
	runtime := newPageRendererRuntime(t)
	data := core.ResourceData{
		Resource: core.Resource{
			ID:    1,
			Title: "<b>Home</b>",
		},
		Values: []core.ResourceFieldValue{
			{
				ResourceID: 1,
				Field:      PageContentFieldCode,
				Value:      "<script>alert(1)</script>",
			},
		},
	}

	html, err := NewPageDefaultRenderer().Render(
		context.Background(),
		runtime,
		data,
	)
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

func TestPageDefaultRendererRendersMainWidgets(t *testing.T) {
	registry := core.NewRuntimeRegistry()
	if err := registry.ForModule("core").Widgets().Register(
		corewidgets.NewTextWidget(),
	); err != nil {
		t.Fatal(err)
	}
	runtime := newPageRendererRuntimeWithRegistry(t, registry)

	html, err := NewPageDefaultRenderer().Render(
		context.Background(),
		runtime,
		core.ResourceData{
			Resource: core.Resource{
				ID:    1,
				Title: "Home",
			},
			Widgets: []core.WidgetInstance{
				{
					ID:       10,
					Widget:   "core.text",
					Template: core.WidgetTemplateDefault,
					Area:     "main",
					Params: core.WidgetParams{
						"text": "<strong>Widget</strong>",
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(
		html,
		"<section>&lt;strong&gt;Widget&lt;/strong&gt;</section>",
	) {
		t.Fatalf("rendered HTML does not contain widget area: %s", html)
	}
}

func newPageRendererRuntime(t *testing.T) *core.SiteRuntime {
	t.Helper()

	return newPageRendererRuntimeWithRegistry(t, core.NewRuntimeRegistry())
}

func newPageRendererRuntimeWithRegistry(
	t *testing.T,
	registry core.Registry,
) *core.SiteRuntime {
	t.Helper()

	app, err := core.NewApp(
		pageRendererCacheManager{},
		pageRendererFileStorageManager{},
		core.NullEventBus{},
		core.NullLogger{},
		pageRendererResourceRepository{},
		pageRendererResourceFieldValueRepository{},
		pageRendererWidgetInstanceRepository{},
	)
	if err != nil {
		t.Fatal(err)
	}

	runtime, err := core.NewSiteRuntime(
		app,
		core.Site{
			ID:          1,
			ProfileCode: "main",
			Locale:      "ru",
		},
		pageRendererSiteProfile{},
		registry,
	)
	if err != nil {
		t.Fatal(err)
	}

	return runtime
}

type pageRendererCacheManager struct{}

func (pageRendererCacheManager) Store(
	name core.CacheStoreName,
) (core.CacheStore, error) {
	return core.NullCacheStore{}, nil
}

func (pageRendererCacheManager) Scope(
	scope core.CacheScope,
) (core.CacheStore, error) {
	return core.NullCacheStore{}, nil
}

type pageRendererFileStorageManager struct{}

func (pageRendererFileStorageManager) Disk(
	name core.FileDisk,
) (core.FileStorage, error) {
	return core.NullFileStorage{}, nil
}

type pageRendererResourceRepository struct{}

func (pageRendererResourceRepository) FindByID(
	ctx context.Context,
	id core.ResourceID,
) (core.Resource, error) {
	return core.Resource{}, nil
}

func (pageRendererResourceRepository) FindByPath(
	ctx context.Context,
	siteID int64,
	path string,
) (core.Resource, error) {
	return core.Resource{}, nil
}

func (pageRendererResourceRepository) FindChildren(
	ctx context.Context,
	parentID core.ResourceID,
) ([]core.Resource, error) {
	return nil, nil
}

type pageRendererResourceFieldValueRepository struct{}

func (pageRendererResourceFieldValueRepository) FindByResourceID(
	ctx context.Context,
	resourceID core.ResourceID,
) ([]core.ResourceFieldValue, error) {
	return nil, nil
}

func (pageRendererResourceFieldValueRepository) FindByResourceAndField(
	ctx context.Context,
	resourceID core.ResourceID,
	field core.ResourceFieldCode,
) (core.ResourceFieldValue, error) {
	return core.ResourceFieldValue{}, nil
}

func (pageRendererResourceFieldValueRepository) Save(
	ctx context.Context,
	value core.ResourceFieldValue,
) (core.ResourceFieldValue, error) {
	return value, nil
}

type pageRendererWidgetInstanceRepository struct{}

func (pageRendererWidgetInstanceRepository) FindForResource(
	ctx context.Context,
	resource core.Resource,
) ([]core.WidgetInstance, error) {
	return nil, nil
}

type pageRendererSiteProfile struct{}

func (pageRendererSiteProfile) Code() string {
	return "main"
}

func (pageRendererSiteProfile) Modules() []core.Module {
	return nil
}
