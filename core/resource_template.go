package core

type ResourceTemplateCode string

type ResourceTemplateDefinition interface {
	Code() ResourceTemplateCode
	Name() string
	ResourceType() ResourceType
}
