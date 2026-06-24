package core

type ResourceTypeDefinition interface {
	Code() ResourceType
	Name() string
}
