package core

import "context"

type SiteResolver interface {
	ResolveByDomain(ctx context.Context, domain string) (Site, error)
}
