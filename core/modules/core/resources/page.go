package resources

import "github.com/vernal96/go-cms/core"

const PageResourceTypeCode core.ResourceType = "page"

type PageResourceType struct{}

func NewPageResourceType() *PageResourceType {
	return &PageResourceType{}
}

func (r *PageResourceType) Code() core.ResourceType {
	return PageResourceTypeCode
}

func (r *PageResourceType) Name() string {
	return "Page"
}

var _ core.ResourceTypeDefinition = (*PageResourceType)(nil)
