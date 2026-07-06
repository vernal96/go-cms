package kernel

type ProfileRegistry interface {
	Register(profile Profile) error
	Get(code ProfileCode) (Profile, bool)
}
