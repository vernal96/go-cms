package memory

import (
	"context"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

const AdapterCode kernel.AdapterCode = "memory"

type Repository struct {
	sitesByDomain map[string]site.Site
}

func NewRepository(sites []site.Site) *Repository {
	sitesByDomain := make(map[string]site.Site, len(sites))

	for _, item := range sites {
		sitesByDomain[item.Domain] = item
	}

	return &Repository{
		sitesByDomain: sitesByDomain,
	}
}

func (r *Repository) FindByDomain(ctx context.Context, domain string) (site.Site, bool, error) {
	_ = ctx

	foundSite, exists := r.sitesByDomain[domain]
	return foundSite, exists, nil
}

var _ site.Repository = (*Repository)(nil)
