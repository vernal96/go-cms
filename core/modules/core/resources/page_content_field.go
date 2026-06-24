package resources

import (
	"github.com/vernal96/go-cms/core"
	corefields "github.com/vernal96/go-cms/core/fields"
	modulefields "github.com/vernal96/go-cms/core/modules/core/fields"
)

const PageContentFieldCode core.ResourceFieldCode = "content"

type PageContentField struct {
	field corefields.FieldType
}

func NewPageContentField() *PageContentField {
	return &PageContentField{
		field: modulefields.NewTextFieldType(),
	}
}

func (f *PageContentField) Code() core.ResourceFieldCode {
	return PageContentFieldCode
}

func (f *PageContentField) Name() string {
	return "Content"
}

func (f *PageContentField) Field() corefields.FieldType {
	return f.field
}

func (f *PageContentField) ResourceType() core.ResourceType {
	return PageResourceTypeCode
}

func (f *PageContentField) ResourceTemplate() core.ResourceTemplateCode {
	return PageDefaultTemplateCode
}

func (f *PageContentField) Required() bool {
	return false
}

var _ core.ResourceFieldDefinition = (*PageContentField)(nil)
