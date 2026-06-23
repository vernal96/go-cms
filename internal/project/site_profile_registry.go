package project

import "github.com/vernal96/go-cms/core"

type SiteProfileRegistration struct {
	Profile core.SiteProfile
}

type SiteProfileRegistry struct {
	profiles []SiteProfileRegistration
}

func NewSiteProfileRegistry() *SiteProfileRegistry {
	return &SiteProfileRegistry{}
}

func (r *SiteProfileRegistry) Register(profile core.SiteProfile) {
	r.profiles = append(r.profiles, SiteProfileRegistration{
		Profile: profile,
	})
}

func (r *SiteProfileRegistry) Registrations() []SiteProfileRegistration {
	return r.profiles
}
