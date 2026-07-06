package kernel

import (
	"errors"
	"fmt"
)

type ProfileRegistryMap struct {
	profiles map[ProfileCode]Profile
}

func NewProfileRegistryMap() *ProfileRegistryMap {
	return &ProfileRegistryMap{
		profiles: make(map[ProfileCode]Profile),
	}
}

func (r *ProfileRegistryMap) Register(profile Profile) error {
	if profile == nil {
		return errors.New("profile is nil")
	}

	code := profile.Code()
	if code == "" {
		return errors.New("profile code is empty")
	}

	if _, exists := r.profiles[code]; exists {
		return fmt.Errorf("profile %q already registered", code)
	}

	r.profiles[code] = profile

	return nil
}

func (r *ProfileRegistryMap) Get(code ProfileCode) (Profile, bool) {
	profile, exists := r.profiles[code]
	return profile, exists
}

var _ ProfileRegistry = (*ProfileRegistryMap)(nil)
