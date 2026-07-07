package kernel

type ProfileCode string

type ProfileModule struct {
	Module Module
	Config any
}

type Profile interface {
	Code() ProfileCode
	Modules() []ProfileModule
}
