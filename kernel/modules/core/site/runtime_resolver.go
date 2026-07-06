package site

import (
	"context"
	"fmt"

	"github.com/vernal96/go-cms/kernel"
)

type RuntimeProvider struct {
	siteResolver   DomainResolver
	profiles       kernel.ProfileRegistry
	runtimeFactory *kernel.SiteRuntimeFactory
}

func NewRuntimeProvider(
	siteResolver DomainResolver,
	profiles kernel.ProfileRegistry,
	runtimeFactory *kernel.SiteRuntimeFactory,
) (*RuntimeProvider, error) {
	if siteResolver == nil {
		return nil, fmt.Errorf("site resolver is nil")
	}

	if profiles == nil {
		return nil, fmt.Errorf("profile registry is nil")
	}

	if runtimeFactory == nil {
		return nil, fmt.Errorf("site runtime factory is nil")
	}

	return &RuntimeProvider{
		siteResolver:   siteResolver,
		profiles:       profiles,
		runtimeFactory: runtimeFactory,
	}, nil
}

func (r *RuntimeProvider) ResolveByDomain(ctx context.Context, domain string) (*kernel.SiteRuntime, bool, error) {
	foundSite, exists, err := r.siteResolver.ResolveByDomain(ctx, domain)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		return nil, false, nil
	}

	profile, exists := r.profiles.Get(foundSite.ProfileCode)
	if !exists {
		return nil, false, fmt.Errorf("profile %q not found", foundSite.ProfileCode)
	}

	runtime, err := r.runtimeFactory.Make(ctx, profile)
	if err != nil {
		return nil, false, err
	}

	return runtime, true, nil
}
