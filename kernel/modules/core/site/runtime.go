package site

import (
	"errors"
	"fmt"

	"github.com/vernal96/go-cms/kernel"
)

type Runtime struct {
	site           Site
	profileRuntime *kernel.ProfileRuntime
}

func NewSiteRuntime(site Site, profileRuntime *kernel.ProfileRuntime) (*Runtime, error) {
	if site.ProfileCode == "" {
		return nil, errors.New("site profile code is empty")
	}

	if profileRuntime == nil {
		return nil, fmt.Errorf("profile runtime is nil")
	}

	runtimeProfileCode := profileRuntime.Profile().Code()
	if site.ProfileCode != runtimeProfileCode {
		return nil, fmt.Errorf(
			"site profile %q does not match profile runtime %q",
			site.ProfileCode,
			runtimeProfileCode,
		)
	}

	return &Runtime{
		site:           site,
		profileRuntime: profileRuntime,
	}, nil
}

func (r *Runtime) Site() Site {
	return r.site
}

func (r *Runtime) Profile() *kernel.ProfileRuntime {
	return r.profileRuntime
}
