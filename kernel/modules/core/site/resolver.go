package site

import (
	"context"
	"errors"
)

type Resolver interface {
	ResolveByDomain(ctx context.Context, domain string) (Site, bool, error)
}

type RepositoryResolver struct {
	repository Repository
}

func NewRepositoryResolver(repository Repository) (*RepositoryResolver, error) {
	if repository == nil {
		return nil, errors.New("site repository is nil")
	}
	return &RepositoryResolver{
		repository: repository,
	}, nil
}

func (r *RepositoryResolver) ResolveByDomain(ctx context.Context, domain string) (Site, bool, error) {
	if domain == "" {
		return Site{}, false, errors.New("site domain is empty")
	}

	return r.repository.FindByDomain(ctx, domain)
}

var _ Resolver = (*RepositoryResolver)(nil)
