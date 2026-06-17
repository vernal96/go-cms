package project

import "github.com/vernal96/go-cms/core"

type SiteProfileRegistration struct {
	Profile core.SiteProfile
}

type SiteProfileConfig struct {
	Profiles []SiteProfileRegistration
}
