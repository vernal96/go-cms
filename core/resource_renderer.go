package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
)

const defaultPageTemplate = `<!doctype html>
<html lang="{{ .Locale }}">
<head>
    <meta charset="utf-8">
    <title>{{ .Title }}</title>
</head>
<body>
    <main>
        <h1>{{ .Title }}</h1>
        <div>{{ .Content }}</div>
    </main>
</body>
</html>`

type ResourceRenderer struct{}

func NewResourceRenderer() *ResourceRenderer {
	return &ResourceRenderer{}
}

func (r *ResourceRenderer) Render(
	ctx context.Context,
	runtime *SiteRuntime,
	data ResourceData,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if runtime == nil {
		return "", errors.New("site runtime is nil")
	}

	resource := data.Resource
	if resource.ID <= 0 {
		return "", errors.New("resource id must be positive")
	}

	if _, exists := runtime.Registry().ResourceTypes().Get(resource.Type); !exists {
		return "", fmt.Errorf("resource type %q is not registered", resource.Type)
	}

	resourceTemplate := ResourceTemplateCode(resource.Template)
	if _, exists := runtime.Registry().ResourceTemplates().Get(
		resource.Type,
		resourceTemplate,
	); !exists {
		return "", fmt.Errorf(
			"resource template %q for resource type %q is not registered",
			resourceTemplate,
			resource.Type,
		)
	}

	if resource.Type != ResourceType("page") ||
		resourceTemplate != ResourceTemplateCode("default") {
		return "", fmt.Errorf(
			"resource renderer does not support resource type %q and template %q",
			resource.Type,
			resourceTemplate,
		)
	}

	values := make(map[ResourceFieldCode]any, len(data.Values))
	for _, value := range data.Values {
		values[value.Field] = value.Value
	}

	content := ""
	if value, exists := values[ResourceFieldCode("content")]; exists {
		if stringValue, ok := value.(string); ok {
			content = stringValue
		} else {
			content = fmt.Sprint(value)
		}
	}

	pageTemplate, err := template.New("page/default").Parse(defaultPageTemplate)
	if err != nil {
		return "", fmt.Errorf("parse default page template: %w", err)
	}

	var output bytes.Buffer
	if err := pageTemplate.Execute(&output, struct {
		Locale  string
		Title   string
		Content string
	}{
		Locale:  runtime.Locale(),
		Title:   resource.Title,
		Content: content,
	}); err != nil {
		return "", fmt.Errorf("render default page template: %w", err)
	}

	return output.String(), nil
}
