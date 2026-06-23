package memorysite

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vernal96/go-cms/core"
)

type Resolver struct {
	sitesByDomain map[string]core.Site
}

func NewResolver(sites ...core.Site) (*Resolver, error) {
	resolver := &Resolver{
		sitesByDomain: make(map[string]core.Site, len(sites)),
	}

	for _, site := range sites {
		domain := normalizeSiteDomain(site.Domain)
		if domain == "" {
			return nil, errors.New("site domain is empty")
		}

		if _, exists := resolver.sitesByDomain[domain]; exists {
			return nil, fmt.Errorf("site domain %q already registered", domain)
		}

		resolver.sitesByDomain[domain] = site
	}

	return resolver, nil
}

func (r *Resolver) ResolveByDomain(ctx context.Context, domain string) (core.Site, error) {
	if err := ctx.Err(); err != nil {
		return core.Site{}, err
	}

	domain = normalizeSiteDomain(domain)
	if domain == "" {
		return core.Site{}, core.ErrSiteNotFound
	}

	site, exists := r.sitesByDomain[domain]
	if !exists {
		return core.Site{}, core.ErrSiteNotFound
	}

	return site, nil
}

func normalizeSiteDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

var _ core.SiteResolver = (*Resolver)(nil)
