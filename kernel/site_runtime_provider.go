package kernel

import (
	"context"
	"fmt"
)

type SiteRuntimeProvider struct {
	profiles       ProfileRegistry
	runtimeFactory *SiteRuntimeFactory
}

func NewSiteRuntimeProvider(
	profiles ProfileRegistry,
	runtimeFactory *SiteRuntimeFactory,
) (*SiteRuntimeProvider, error) {
	if profiles == nil {
		return nil, fmt.Errorf("profile registry is nil")
	}

	if runtimeFactory == nil {
		return nil, fmt.Errorf("site runtime factory is nil")
	}

	return &SiteRuntimeProvider{
		profiles:       profiles,
		runtimeFactory: runtimeFactory,
	}, nil
}

func (p *SiteRuntimeProvider) RuntimeByProfileCode(ctx context.Context, profileCode ProfileCode) (*SiteRuntime, error) {
	if profileCode == "" {
		return nil, fmt.Errorf("profile code is empty")
	}

	profile, exists := p.profiles.Get(profileCode)
	if !exists {
		return nil, fmt.Errorf("profile %q not found", profileCode)
	}

	return p.runtimeFactory.Make(ctx, profile)
}
