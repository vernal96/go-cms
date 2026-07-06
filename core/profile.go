package core

// Profile describes a CMS site profile.
//
// A profile is code-level configuration for a type of site. Multiple site
// records can use the same profile, but the profile itself is defined in Go.
type Profile interface {
	// Code returns the stable profile code.
	Code() string
}
