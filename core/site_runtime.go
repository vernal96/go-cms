package core

import (
	"errors"
	"fmt"
)

type SiteRuntime struct {
	site     Site
	profile  SiteProfile
	registry Registry
}

func NewSiteRuntime(site Site, profile SiteProfile, registry Registry) (*SiteRuntime, error) {
	if site.ProfileCode == "" {
		return nil, errors.New("site code is empty")
	}

	if profile == nil {
		return nil, errors.New("site profile is nil")
	}

	if profile.Code() != site.ProfileCode {
		return nil, fmt.Errorf(
			"site profile code mismatch: site code %q, profile code %q",
			site.ProfileCode,
			profile.Code(),
		)
	}

	if registry == nil {
		registry = NewDefaultRegistry()
	}

	return &SiteRuntime{
		site:     site,
		profile:  profile,
		registry: registry,
	}, nil
}

func (r *SiteRuntime) Site() Site {
	return r.site
}

func (r *SiteRuntime) Profile() SiteProfile {
	return r.profile
}

func (r *SiteRuntime) Registry() Registry {
	return r.registry
}

func (r *SiteRuntime) Locale() string {
	return r.site.Locale
}
