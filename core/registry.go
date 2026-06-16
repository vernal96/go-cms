package core

type Registry interface{}

type DefaultRegistry struct{}

func NewDefaultRegistry() *DefaultRegistry {
	return &DefaultRegistry{}
}
