package kernel

type ProfileCode string

type ProfileConfig struct {
}

type Profile interface {
	Code() ProfileCode
	Modules() []Module
}
