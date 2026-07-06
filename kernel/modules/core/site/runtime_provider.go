package site

import (
	"context"
	"fmt"

	"github.com/vernal96/go-cms/kernel"
)

type RuntimeProvider struct {
	domainResolver      DomainResolver
	siteRuntimeProvider *kernel.SiteRuntimeProvider
}

func NewRuntimeProvider(
	domainResolver DomainResolver,
	siteRuntimeProvider *kernel.SiteRuntimeProvider,
) (*RuntimeProvider, error) {
	if domainResolver == nil {
		return nil, fmt.Errorf("domain resolver is nil")
	}

	if siteRuntimeProvider == nil {
		return nil, fmt.Errorf("site runtime provider is nil")
	}

	return &RuntimeProvider{
		domainResolver:      domainResolver,
		siteRuntimeProvider: siteRuntimeProvider,
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

	runtime, err := p.siteRuntimeProvider.RuntimeByProfileCode(ctx, foundSite.ProfileCode)
	if err != nil {
		return nil, false, err
	}

	return runtime, true, nil
}
