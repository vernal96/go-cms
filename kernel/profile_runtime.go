package kernel

type ProfileRuntime struct {
	app      *App
	profile  Profile
	registry Registry
}

func NewProfileRuntime(
	app *App,
	profile Profile,
	registry Registry,
) *ProfileRuntime {
	return &ProfileRuntime{
		app:      app,
		profile:  profile,
		registry: registry,
	}
}

func (r *ProfileRuntime) App() *App {
	return r.app
}

func (r *ProfileRuntime) Profile() Profile {
	return r.profile
}

func (r *ProfileRuntime) Registry() Registry {
	return r.registry
}
