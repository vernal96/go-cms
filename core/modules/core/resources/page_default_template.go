package resources

import "github.com/vernal96/go-cms/core"

const PageDefaultTemplateCode core.ResourceTemplateCode = "default"

type PageDefaultTemplate struct{}

func NewPageDefaultTemplate() *PageDefaultTemplate {
	return &PageDefaultTemplate{}
}

func (t *PageDefaultTemplate) Code() core.ResourceTemplateCode {
	return PageDefaultTemplateCode
}

func (t *PageDefaultTemplate) Name() string {
	return "Default page"
}

func (t *PageDefaultTemplate) ResourceType() core.ResourceType {
	return PageResourceTypeCode
}

var _ core.ResourceTemplateDefinition = (*PageDefaultTemplate)(nil)
