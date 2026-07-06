package site

import (
	"context"
	"fmt"

	"github.com/vernal96/go-cms/kernel"
)

type RuntimeProvider struct {
	domainResolver DomainResolver
	profiles       kernel.ProfileRegistry
	runtimeFactory *kernel.SiteRuntimeFactory
}

func NewRuntimeProvider(
	domainResolver DomainResolver,
	profiles kernel.ProfileRegistry,
	runtimeFactory *kernel.SiteRuntimeFactory,
) (*RuntimeProvider, error) {
	if domainResolver == nil {
		return nil, fmt.Errorf("domain resolver is nil")
	}

	if profiles == nil {
		return nil, fmt.Errorf("profile registry is nil")
	}

	if runtimeFactory == nil {
		return nil, fmt.Errorf("site runtime factory is nil")
	}

	return &RuntimeProvider{
		domainResolver: domainResolver,
		profiles:       profiles,
		runtimeFactory: runtimeFactory,
	}, nil
}

func (p *RuntimeProvider) RuntimeByDomain(ctx context.Context, domain string) (*kernel.SiteRuntime, bool, error) {
	foundSite, exists, err := p.domainResolver.ResolveByDomain(ctx, domain)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		return nil, false, nil
	}

	profile, exists := p.profiles.Get(foundSite.ProfileCode)
	if !exists {
		return nil, false, fmt.Errorf("profile %q not found", foundSite.ProfileCode)
	}

	runtime, err := p.runtimeFactory.Make(ctx, profile)
	if err != nil {
		return nil, false, err
	}

	return runtime, true, nil
}
