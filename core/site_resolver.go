package core

import (
	"context"
	"errors"
)

var ErrSiteNotFound = errors.New("site not found")

type SiteResolver interface {
	ResolveByDomain(ctx context.Context, domain string) (Site, error)
}
