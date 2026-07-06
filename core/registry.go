package core

type Registry interface {
	ForModule(moduleCode ModuleCode) Registry
}
