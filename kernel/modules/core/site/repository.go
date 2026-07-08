package site

import (
	"context"

	"github.com/vernal96/go-cms/kernel"
)

const RepositoryAdapterContract kernel.AdapterContractCode = "core.site.repository"

type Repository interface {
	FindByDomain(ctx context.Context, domain string) (Site, bool, error)
}
