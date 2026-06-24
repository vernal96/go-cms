package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/vernal96/go-cms/core"
)

const (
	ResourceRoutePath           = "/_cms/resource"
	ResourceFieldValueRoutePath = "/_cms/resource/field-value"
)

type ResourceController struct {
	reader *core.ResourceReader
	writer *core.ResourceFieldValueWriter
}

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

type saveResourceFieldValueRequest struct {
	Path  string                 `json:"path"`
	Field core.ResourceFieldCode `json:"field"`
	Value any                    `json:"value"`
}

func NewResourceController() *ResourceController {
	return &ResourceController{
		reader: core.NewResourceReader(),
		writer: core.NewResourceFieldValueWriter(),
	}
}

func (c *ResourceController) Routes() []core.Route {
	return []core.Route{
		{
			Method:  core.RouteMethodGet,
			Path:    ResourceRoutePath,
			Handler: c.resource,
		},
		{
			Method:  core.RouteMethodPost,
			Path:    ResourceFieldValueRoutePath,
			Handler: c.saveResourceFieldValue,
		},
	}
}

func (c *ResourceController) resource(
	ctx context.Context,
	runtime *core.SiteRuntime,
	request *http.Request,
) (any, error) {
	resourcePath := request.URL.Query().Get("path")
	if resourcePath == "" {
		resourcePath = "/"
	}

	data, err := c.reader.ReadByPath(ctx, runtime, resourcePath)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"resource":          data.Resource,
		"registered_fields": resourceFieldDefinitionsResponse(data.Fields),
		"field_values":      resourceFieldValuesResponse(data.Values),
	}, nil
}

func (c *ResourceController) saveResourceFieldValue(
	ctx context.Context,
	runtime *core.SiteRuntime,
	request *http.Request,
) (any, error) {
	input, err := decodeSaveResourceFieldValueRequest(request)
	if err != nil {
		return nil, err
	}

	if input.Path == "" {
		input.Path = "/"
	}

	data, err := c.reader.ReadByPath(ctx, runtime, input.Path)
	if err != nil {
		return nil, err
	}

	savedValue, err := c.writer.Save(
		ctx,
		runtime,
		data.Resource,
		input.Field,
		input.Value,
	)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"field_value": resourceFieldValueResponseFromValue(savedValue),
	}, nil
}

func decodeSaveResourceFieldValueRequest(
	request *http.Request,
) (saveResourceFieldValueRequest, error) {
	var input saveResourceFieldValueRequest
	if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
		return saveResourceFieldValueRequest{}, fmt.Errorf(
			"decode resource field value request: %w",
			err,
		)
	}

	return input, nil
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
		response = append(response, resourceFieldValueResponseFromValue(value))
	}

	return response
}

func resourceFieldValueResponseFromValue(
	value core.ResourceFieldValue,
) resourceFieldValueResponse {
	return resourceFieldValueResponse{
		Field: value.Field,
		Value: value.Value,
	}
}

var _ core.Controller = (*ResourceController)(nil)
