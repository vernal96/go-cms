package site_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

type repositoryStub struct {
	findByDomain func(ctx context.Context, domain string) (site.Site, bool, error)
}

func (r *repositoryStub) FindByDomain(ctx context.Context, domain string) (site.Site, bool, error) {
	return r.findByDomain(ctx, domain)
}

func TestNewRepositoryDomainResolver_RejectsNilRepository(t *testing.T) {
	resolver, err := site.NewRepositoryDomainResolver(nil)

	if resolver != nil {
		t.Fatal("expected nil resolver")
	}

	if err == nil {
		t.Fatal("expected nil repository error")
	}
}

func TestRepositoryDomainResolver_RejectsEmptyDomain(t *testing.T) {
	repositoryCalled := false
	repository := &repositoryStub{
		findByDomain: func(ctx context.Context, domain string) (site.Site, bool, error) {
			repositoryCalled = true
			return site.Site{}, false, nil
		},
	}

	resolver, err := site.NewRepositoryDomainResolver(repository)
	if err != nil {
		t.Fatalf("create domain resolver: %v", err)
	}

	_, _, err = resolver.ResolveByDomain(context.Background(), "")
	if err == nil {
		t.Fatal("expected empty domain error")
	}

	if repositoryCalled {
		t.Fatal("repository was called for an empty domain")
	}
}

func TestRepositoryDomainResolver_DelegatesToRepository(t *testing.T) {
	type contextKey string

	ctx := context.WithValue(context.Background(), contextKey("request"), "test")
	wantDomain := "example.com"
	wantSite := site.Site{
		ID:          10,
		ProfileCode: "public",
		Domain:      wantDomain,
		Locale:      "ru-RU",
		Settings: map[string]any{
			"theme": "default",
		},
	}

	repository := &repositoryStub{
		findByDomain: func(gotContext context.Context, gotDomain string) (site.Site, bool, error) {
			if gotContext != ctx {
				t.Fatal("repository received a different context")
			}

			if gotDomain != wantDomain {
				t.Fatalf("unexpected domain: got %q, want %q", gotDomain, wantDomain)
			}

			return wantSite, true, nil
		},
	}

	resolver, err := site.NewRepositoryDomainResolver(repository)
	if err != nil {
		t.Fatalf("create domain resolver: %v", err)
	}

	gotSite, exists, err := resolver.ResolveByDomain(ctx, wantDomain)
	if err != nil {
		t.Fatalf("resolve site: %v", err)
	}

	if !exists {
		t.Fatal("expected site to exist")
	}

	if !reflect.DeepEqual(gotSite, wantSite) {
		t.Fatalf("unexpected site: got %#v, want %#v", gotSite, wantSite)
	}
}

func TestRepositoryDomainResolver_ReturnsRepositoryError(t *testing.T) {
	repositoryErr := errors.New("repository failed")
	repository := &repositoryStub{
		findByDomain: func(ctx context.Context, domain string) (site.Site, bool, error) {
			return site.Site{}, false, repositoryErr
		},
	}

	resolver, err := site.NewRepositoryDomainResolver(repository)
	if err != nil {
		t.Fatalf("create domain resolver: %v", err)
	}

	_, _, err = resolver.ResolveByDomain(context.Background(), "example.com")
	if !errors.Is(err, repositoryErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}

