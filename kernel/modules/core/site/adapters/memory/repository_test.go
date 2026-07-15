package memory_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/site/adapters/memory"
)

func TestRepository_FindByDomainReturnsSite(t *testing.T) {
	wantSite := site.Site{
		ID:          10,
		ProfileCode: "public",
		Domain:      "example.com",
		Locale:      "ru-RU",
		Settings: map[string]any{
			"theme": "default",
		},
	}

	repository := memory.NewRepository([]site.Site{wantSite})
	gotSite, exists, err := repository.FindByDomain(context.Background(), wantSite.Domain)
	if err != nil {
		t.Fatalf("find site by domain: %v", err)
	}

	if !exists {
		t.Fatal("expected site to exist")
	}

	if !reflect.DeepEqual(gotSite, wantSite) {
		t.Fatalf("unexpected site: got %#v, want %#v", gotSite, wantSite)
	}
}

func TestRepository_FindByDomainReturnsNotFound(t *testing.T) {
	repository := memory.NewRepository(nil)

	foundSite, exists, err := repository.FindByDomain(context.Background(), "missing.example.com")
	if err != nil {
		t.Fatalf("find site by domain: %v", err)
	}

	if exists {
		t.Fatal("unexpected site found")
	}

	if !reflect.DeepEqual(foundSite, site.Site{}) {
		t.Fatalf("expected zero site, got %#v", foundSite)
	}
}

