package core

import "github.com/vernal96/go-cms/core/fields"

type ResourceFieldCode string

type ResourceFieldDefinition interface {
	Code() ResourceFieldCode
	Name() string
	Field() fields.FieldType
	ResourceType() ResourceType
	ResourceTemplate() ResourceTemplateCode
	Required() bool
}
