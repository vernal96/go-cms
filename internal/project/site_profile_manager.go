package project

import (
	"errors"
	"fmt"

	"github.com/vernal96/go-cms/core"
)

type SiteProfileManager struct {
	profiles map[string]core.SiteProfile
}

func NewSiteProfileManager(profilesRegistry *SiteProfileRegistry) (*SiteProfileManager, error) {
	registrations := profilesRegistry.Registrations()
	profiles := make(map[string]core.SiteProfile, len(registrations))

	for _, registration := range registrations {
		profile := registration.Profile
		if profile == nil {
			return nil, errors.New("site profile is nil")
		}

		code := profile.Code()
		if code == "" {
			return nil, errors.New("site profile code is empty")
		}

		if _, exists := profiles[code]; exists {
			return nil, fmt.Errorf("site profile %q is already registered", code)
		}

		profiles[code] = profile
	}

	return &SiteProfileManager{
		profiles: profiles,
	}, nil
}

func (m *SiteProfileManager) Profile(code string) (core.SiteProfile, bool) {
	profile, exists := m.profiles[code]
	return profile, exists
}

func (m *SiteProfileManager) Profiles() []core.SiteProfile {
	result := make([]core.SiteProfile, 0, len(m.profiles))

	for _, profile := range m.profiles {
		result = append(result, profile)
	}

	return result
}

var _ core.SiteProfileRegistry = (*SiteProfileManager)(nil)
