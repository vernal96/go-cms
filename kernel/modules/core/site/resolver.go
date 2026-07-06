package site

import (
	"context"
	"errors"
)

type DomainResolver interface {
	ResolveByDomain(ctx context.Context, domain string) (Site, bool, error)
}

type RepositoryDomainResolver struct {
	repository Repository
}

func NewRepositoryDomainResolver(repository Repository) (*RepositoryDomainResolver, error) {
	if repository == nil {
		return nil, errors.New("site repository is nil")
	}

	return &RepositoryDomainResolver{
		repository: repository,
	}, nil
}

func (r *RepositoryDomainResolver) ResolveByDomain(ctx context.Context, domain string) (Site, bool, error) {
	if domain == "" {
		return Site{}, false, errors.New("site domain is empty")
	}

	return r.repository.FindByDomain(ctx, domain)
}

var _ DomainResolver = (*RepositoryDomainResolver)(nil)
