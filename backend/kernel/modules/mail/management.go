package mail

import (
	"context"
	"errors"

	"github.com/vernal96/go-cms/kernel/modules/core/group"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/security"
)

type SiteCatalog interface {
	RuntimeByID(site.ID) (*site.Runtime, bool)
}

type SiteAccessPolicy interface {
	Check(context.Context, security.Actor, site.ID, group.SiteAccessAction) error
}

type Management struct {
	sites  SiteCatalog
	policy SiteAccessPolicy
}

func NewManagement(sites SiteCatalog, policy SiteAccessPolicy) (*Management, error) {
	if sites == nil || policy == nil {
		return nil, errors.New("mail management dependencies are nil")
	}
	return &Management{sites: sites, policy: policy}, nil
}

func (m *Management) Service(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	action group.SiteAccessAction,
) (*Service, error) {
	if ctx == nil {
		return nil, errors.New("mail management context is nil")
	}
	if siteID <= 0 {
		return nil, ErrNotFound
	}
	if err := m.policy.Check(ctx, actor, siteID, action); err != nil {
		return nil, err
	}
	runtime, exists := m.sites.RuntimeByID(siteID)
	if !exists || runtime == nil || runtime.Profile() == nil {
		return nil, ErrNotFound
	}
	moduleRuntime, exists := runtime.Profile().Registry().Module(ModuleCode)
	if !exists {
		return nil, ErrNotFound
	}
	provider, ok := moduleRuntime.(interface{ Mail() *Service })
	if !ok || provider.Mail() == nil {
		return nil, errors.New("site mail runtime is invalid")
	}
	return provider.Mail(), nil
}
