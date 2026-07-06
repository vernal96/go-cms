package core

type Registry interface {
	ForModule(moduleCode ModuleCode) Registry
}

type RuntimeRegistry struct {
	ModuleCode ModuleCode
}

func NewRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{}
}

func (r *RuntimeRegistry) ForModule(moduleCode ModuleCode) Registry {
	return &RuntimeRegistry{
		ModuleCode: moduleCode,
	}
}
