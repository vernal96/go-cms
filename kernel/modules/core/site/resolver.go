package site

import "context"

type Resolver interface {
	ResolveByDomain(ctx context.Context, domain string) (Site, bool, error)
}

type RepositoryResolver struct {
	repository Repository
}

func NewRepositoryResolver(repository Repository) *RepositoryResolver {
	return &RepositoryResolver{
		repository: repository,
	}
}

func (r *RepositoryResolver) ResolveByDomain(ctx context.Context, domain string) (Site, bool, error) {
	return r.repository.FindByDomain(ctx, domain)
}

var _ Resolver = (*RepositoryResolver)(nil)
