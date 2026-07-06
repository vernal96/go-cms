package site

import "context"

type Repository interface {
	FindByDomain(ctx context.Context, domain string) (Site, bool, error)
}
