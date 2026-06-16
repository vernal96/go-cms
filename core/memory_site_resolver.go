package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type MemorySiteResolver struct {
	sitesByDomain map[string]Site
}

func NewMemorySiteResolver(sites ...Site) (*MemorySiteResolver, error) {
	resolver := &MemorySiteResolver{
		sitesByDomain: make(map[string]Site, len(sites)),
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

func (r *MemorySiteResolver) ResolveByDomain(ctx context.Context, domain string) (Site, error) {
	if err := ctx.Err(); err != nil {
		return Site{}, err
	}

	domain = normalizeSiteDomain(domain)
	if domain == "" {
		return Site{}, ErrSiteNotFound
	}

	site, exists := r.sitesByDomain[domain]
	if !exists {
		return Site{}, ErrSiteNotFound
	}

	return site, nil
}

func normalizeSiteDomain(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}
