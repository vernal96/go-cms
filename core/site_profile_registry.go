package core

import (
	"errors"
	"fmt"
)

type SiteProfileRegistry interface {
	RegisterProfile(profile SiteProfile) error
	Profile(code string) (SiteProfile, bool)
	Profiles() []SiteProfile
}

type DefaultSiteProfileRegistry struct {
	profiles map[string]SiteProfile
}

func NewDefaultSiteProfileRegistry() *DefaultSiteProfileRegistry {
	return &DefaultSiteProfileRegistry{
		profiles: make(map[string]SiteProfile),
	}
}

func (r *DefaultSiteProfileRegistry) RegisterProfile(profile SiteProfile) error {
	if profile == nil {
		return errors.New("site profile is nil")
	}

	code := profile.Code()

	if code == "" {
		return errors.New("site profile code is empty")
	}

	if _, exists := r.profiles[code]; exists {
		return fmt.Errorf("site profile %q already registered", code)
	}

	r.profiles[code] = profile

	return nil
}

func (r *DefaultSiteProfileRegistry) Profile(code string) (SiteProfile, bool) {
	profile, exists := r.profiles[code]
	return profile, exists
}

func (r *DefaultSiteProfileRegistry) Profiles() []SiteProfile {
	result := make([]SiteProfile, 0, len(r.profiles))

	for _, profiles := range r.profiles {
		result = append(result, profiles)
	}

	return result
}
