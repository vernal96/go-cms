package core

import (
	"context"
	"errors"
)

var ErrSiteNotFound = errors.New("site not found")

type SiteRepository interface {
	Migrate(ctx context.Context) error
	FindByDomain(ctx context.Context, domain string) (Site, error)
}
