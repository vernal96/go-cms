package site

import (
	"context"
	"fmt"

	"github.com/vernal96/go-cms/kernel"
)

type RuntimeProvider struct {
	domainResolver DomainResolver
	app            *kernel.App
}

func NewRuntimeProvider(
	domainResolver DomainResolver,
	app *kernel.App,
) (*RuntimeProvider, error) {
	if domainResolver == nil {
		return nil, fmt.Errorf("domain resolver is nil")
	}

	if app == nil {
		return nil, fmt.Errorf("app is nil")
	}

	return &RuntimeProvider{
		domainResolver: domainResolver,
		app:            app,
	}, nil
}

func (p *RuntimeProvider) RuntimeByDomain(
	ctx context.Context,
	domain string,
) (*Runtime, bool, error) {
	foundSite, exists, err := p.domainResolver.ResolveByDomain(
		ctx,
		domain,
	)
	if err != nil {
		return nil, false, err
	}

	if !exists {
		return nil, false, nil
	}

	profileRuntime, exists := p.app.ProfileRuntime(
		foundSite.ProfileCode,
	)
	if !exists {
		return nil, false, fmt.Errorf(
			"profile runtime %q not found",
			foundSite.ProfileCode,
		)
	}

	runtime, err := NewSiteRuntime(
		foundSite,
		profileRuntime,
	)
	if err != nil {
		return nil, false, err
	}

	return runtime, true, nil
}
