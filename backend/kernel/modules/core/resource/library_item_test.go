package resource

import (
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
)

func TestEffectiveLibraryItemURLUsesCurrentLibraryPath(t *testing.T) {
	firstPath := "/news"
	library := Resource{
		Path: &firstPath,
		TypeSettings: map[string]any{
			"item_url_pattern": "/{year}/{month}/{slug}",
		},
	}
	publishedAt := time.Date(2026, time.August, 22, 12, 0, 0, 0, time.UTC)
	item := LibraryItem{ID: 42, Slug: "release", PublishedAt: &publishedAt}

	url, err := EffectiveLibraryItemURL(library, item)
	if err != nil || url != "/news/2026/08/release" {
		t.Fatalf("first effective URL = %q, %v", url, err)
	}
	movedPath := "/company/updates"
	library.Path = &movedPath
	url, err = EffectiveLibraryItemURL(library, item)
	if err != nil || url != "/company/updates/2026/08/release" {
		t.Fatalf("moved effective URL = %q, %v", url, err)
	}
	key, matched := MatchLibraryItemPattern(
		library.TypeSettings["item_url_pattern"].(string),
		"/2026/08/release",
	)
	if !matched || key.Slug != item.Slug || key.ID != 0 {
		t.Fatalf("matched key = %#v, matched = %t", key, matched)
	}

	library.TypeSettings["item_url_pattern"] = resourcetype.DefaultItemURLPattern
	url, err = EffectiveLibraryItemURL(library, item)
	if err != nil || url != "/company/updates/release" {
		t.Fatalf("default-pattern URL = %q, %v", url, err)
	}
}
