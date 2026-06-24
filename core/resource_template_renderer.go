package core

import "context"

type ResourceTemplateRenderer interface {
	ResourceType() ResourceType
	ResourceTemplate() ResourceTemplateCode
	Render(
		ctx context.Context,
		runtime *SiteRuntime,
		data ResourceData,
	) (string, error)
}
