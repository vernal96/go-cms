package controllers

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/vernal96/go-cms/core"
)

const ResourceRoutePath = "/_cms/resource"

type ResourceController struct{}

type resourceFieldDefinitionResponse struct {
	Code             core.ResourceFieldCode    `json:"code"`
	Name             string                    `json:"name"`
	FieldType        string                    `json:"field_type"`
	ResourceType     core.ResourceType         `json:"resource_type"`
	ResourceTemplate core.ResourceTemplateCode `json:"resource_template"`
	Required         bool                      `json:"required"`
}

type resourceFieldValueResponse struct {
	Field core.ResourceFieldCode `json:"field"`
	Value any                    `json:"value"`
}

func NewResourceController() *ResourceController {
	return &ResourceController{}
}

func (c *ResourceController) Routes() []core.Route {
	return []core.Route{
		{
			Method:  core.RouteMethodGet,
			Path:    ResourceRoutePath,
			Handler: c.resource,
		},
	}
}

func (c *ResourceController) resource(
	ctx context.Context,
	runtime *core.SiteRuntime,
	request *http.Request,
) (any, error) {
	if runtime == nil {
		return nil, errors.New("site runtime is nil")
	}

	resourcePath := request.URL.Query().Get("path")
	if resourcePath == "" {
		resourcePath = "/"
	}
	if !strings.HasPrefix(resourcePath, "/") {
		return nil, errors.New("resource path must start with /")
	}

	resource, err := runtime.App().Resources().FindByPath(ctx, runtime.Site().ID, resourcePath)
	if err != nil {
		return nil, err
	}

	registeredFields := runtime.Registry().ResourceFields().AllForTemplate(
		resource.Type,
		core.ResourceTemplateCode(resource.Template),
	)

	fieldValues, err := runtime.App().ResourceFieldValues().FindByResourceID(ctx, resource.ID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"resource":          resource,
		"registered_fields": resourceFieldDefinitionsResponse(registeredFields),
		"field_values":      resourceFieldValuesResponse(fieldValues),
	}, nil
}

func resourceFieldDefinitionsResponse(
	fields []core.ResourceFieldDefinition,
) []resourceFieldDefinitionResponse {
	response := make([]resourceFieldDefinitionResponse, 0, len(fields))

	for _, field := range fields {
		response = append(response, resourceFieldDefinitionResponse{
			Code:             field.Code(),
			Name:             field.Name(),
			FieldType:        string(field.Field().Code()),
			ResourceType:     field.ResourceType(),
			ResourceTemplate: field.ResourceTemplate(),
			Required:         field.Required(),
		})
	}

	return response
}

func resourceFieldValuesResponse(
	values []core.ResourceFieldValue,
) []resourceFieldValueResponse {
	response := make([]resourceFieldValueResponse, 0, len(values))

	for _, value := range values {
		response = append(response, resourceFieldValueResponse{
			Field: value.Field,
			Value: value.Value,
		})
	}

	return response
}

var _ core.Controller = (*ResourceController)(nil)
