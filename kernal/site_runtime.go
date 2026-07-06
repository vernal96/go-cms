package kernal

type SiteRuntime struct {
	app      *App
	profile  Profile
	registry Registry
}

func NewSiteRuntime(app *App, profile Profile, registry Registry) *SiteRuntime {
	return &SiteRuntime{
		app:      app,
		profile:  profile,
		registry: registry,
	}
}

func (r *SiteRuntime) App() *App {
	return r.app
}

func (r *SiteRuntime) Profile() Profile {
	return r.profile
}

func (r *SiteRuntime) Registry() Registry {
	return r.registry
}
