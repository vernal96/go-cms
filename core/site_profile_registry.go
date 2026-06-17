package core

type SiteProfileRegistry interface {
	Profile(code string) (SiteProfile, bool)
	Profiles() []SiteProfile
}
