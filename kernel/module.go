package kernel

type ModuleCode string

type Module interface {
	Code() ModuleCode
}
