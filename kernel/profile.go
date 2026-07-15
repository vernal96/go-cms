package kernel

type ProfileCode string

type ProfileModule struct {
	Module       Module
	ModuleConfig any
}

type Profile interface {
	Code() ProfileCode
	Modules() []ProfileModule
}
