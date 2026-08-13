package seo

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/security"
)

func testRenderer(t *testing.T, resultLimit int) *Renderer {
	t.Helper()
	renderer, err := NewRenderer(kernel.Profile{
		Params: []field.Definition{{Key: "site_name"}},
		Templates: []template.Definition{{Fields: []field.Definition{{
			Key: "subtitle",
		}}}},
	}, 1000, resultLimit)
	if err != nil {
		t.Fatal(err)
	}
	return renderer
}

func TestRendererSubstitutesEveryVariableAndStripsHTML(t *testing.T) {
	t.Parallel()
	renderer := testRenderer(t, 1000)
	path := "/contacts"
	result, err := renderer.Render(Settings{
		TitleTemplate: "{{ resource.title }} | {{ resource.menu_title }} | " +
			"{{ resource.annotation }} | {{ resource.slug }} | {{ resource.path }} | " +
			"{{ resource.field.subtitle }} | {{ site.domain }} | {{ site.locale }} | " +
			"{{ site.field.site_name }}",
		DescriptionTemplate: "<b>{{ resource.annotation }}</b>",
		KeywordsTemplate:    "company, {{ resource.slug }}",
		CanonicalTemplate:   "https://{{ site.domain }}{{ resource.path }}",
		RobotsIndex:         true,
		RobotsFollow:        true,
	}, RenderInput{
		Site: site.Site{
			Domain: "example.com", Locale: "ru-RU",
			Settings: map[string]any{"site_name": "<b>Компания</b>"},
		},
		Resource: resource.Resource{
			Title: "<strong>Контакты</strong>", MenuTitle: "Меню",
			Annotation: "Свяжитесь", Slug: "contacts", Path: &path,
			Settings: map[string]any{"subtitle": "О нас"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantTitle := "Контакты | Меню | Свяжитесь | contacts | /contacts | О нас | example.com | ru-RU | Компания"
	if result.Title != wantTitle || result.Description != "Свяжитесь" {
		t.Fatalf("rendered = %#v", result)
	}
	if len(result.Keywords) != 2 || result.Keywords[1] != "contacts" ||
		result.CanonicalURL != "https://example.com/contacts" {
		t.Fatalf("normalized values = %#v", result.PublicData)
	}
	if result.OpenGraph.Title != result.Title ||
		result.OpenGraph.Description != result.Description {
		t.Fatalf("Open Graph fallback = %#v", result.OpenGraph)
	}
}

func TestRendererWarnsForEmptyValuesAndForcesPreviewNoindex(t *testing.T) {
	t.Parallel()
	renderer := testRenderer(t, 100)
	result, err := renderer.Render(Settings{
		TitleTemplate:       "{{ resource.title }}",
		DescriptionTemplate: "{{ resource.annotation }}",
		RobotsIndex:         true,
		RobotsFollow:        true,
	}, RenderInput{Preview: true, Resource: resource.Resource{Title: "Page"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Robots.Index || !result.Robots.Follow || len(result.Warnings) != 1 ||
		result.Warnings[0].Variable != "resource.annotation" {
		t.Fatalf("preview = %#v", result)
	}
}

func TestRendererRejectsUnknownVariablesAndLongResults(t *testing.T) {
	t.Parallel()
	renderer := testRenderer(t, 5)
	if err := renderer.Validate(Settings{TitleTemplate: "{{ resource.unknown }}"}); err == nil {
		t.Fatal("unknown variable was accepted")
	}
	_, err := renderer.Render(Settings{
		TitleTemplate:       "{{ resource.title }}",
		DescriptionTemplate: "ok",
	}, RenderInput{Resource: resource.Resource{Title: "123456"}})
	var validation ValidationError
	if !errors.As(err, &validation) || validation.Field != "title_template" ||
		!strings.Contains(err.Error(), "exceeds 5") {
		t.Fatalf("result length error = %v", err)
	}
}

type memoryRepository struct {
	metadata seoMetadataHolder
}

type seoMetadataHolder struct {
	value  Metadata
	exists bool
}

func (r *memoryRepository) ByResource(
	_ context.Context,
	siteID site.ID,
	resourceID resource.ID,
) (Metadata, error) {
	if !r.metadata.exists || r.metadata.value.SiteID != siteID ||
		r.metadata.value.ResourceID != resourceID {
		return Metadata{}, ErrNotFound
	}
	return r.metadata.value, nil
}

func (r *memoryRepository) Save(_ context.Context, metadata Metadata) (Metadata, error) {
	r.metadata = seoMetadataHolder{value: metadata, exists: true}
	return metadata, nil
}

func TestServiceUsesDefaultsValidatesSaveAndKeepsSiteScope(t *testing.T) {
	t.Parallel()
	repository := &memoryRepository{}
	renderer := testRenderer(t, 100)
	defaults := Settings{
		TitleTemplate:       "{{ resource.title }}",
		DescriptionTemplate: "{{ resource.annotation }}",
		RobotsIndex:         true,
		RobotsFollow:        true,
	}
	service, err := NewService(repository, renderer, defaults)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := service.Get(context.Background(), 7, 9)
	if err != nil || loaded != defaults {
		t.Fatalf("defaults = %#v, %v", loaded, err)
	}
	saved, err := service.Save(
		context.Background(), security.User(3), 7, 9,
		Settings{RobotsFollow: true},
	)
	if err != nil || saved.TitleTemplate != defaults.TitleTemplate ||
		saved.DescriptionTemplate != defaults.DescriptionTemplate ||
		repository.metadata.value.UpdatedBy == nil ||
		*repository.metadata.value.UpdatedBy != 3 {
		t.Fatalf("saved defaults = %#v, %#v, %v", saved, repository.metadata, err)
	}
	if _, err := service.Get(context.Background(), 8, 9); err != nil {
		t.Fatalf("other site must receive its defaults: %v", err)
	}
	if _, err := service.Save(
		context.Background(), security.User(3), 7, 9,
		Settings{TitleTemplate: "{{ resource.content }}"},
	); err == nil {
		t.Fatal("invalid template was saved")
	}
}
