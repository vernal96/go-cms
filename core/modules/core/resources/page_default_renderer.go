package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"

	"github.com/vernal96/go-cms/core"
)

const defaultPageHTML = `<!doctype html>
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

type PageDefaultRenderer struct{}

func NewPageDefaultRenderer() *PageDefaultRenderer {
	return &PageDefaultRenderer{}
}

func (r *PageDefaultRenderer) ResourceType() core.ResourceType {
	return PageResourceTypeCode
}

func (r *PageDefaultRenderer) ResourceTemplate() core.ResourceTemplateCode {
	return PageDefaultTemplateCode
}

func (r *PageDefaultRenderer) Render(
	ctx context.Context,
	runtime *core.SiteRuntime,
	data core.ResourceData,
) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	if runtime == nil {
		return "", errors.New("site runtime is nil")
	}

	values := make(map[core.ResourceFieldCode]any, len(data.Values))
	for _, value := range data.Values {
		values[value.Field] = value.Value
	}

	content := ""
	if value, exists := values[PageContentFieldCode]; exists {
		if stringValue, ok := value.(string); ok {
			content = stringValue
		} else {
			content = fmt.Sprint(value)
		}
	}

	pageTemplate, err := template.New("page/default").Parse(defaultPageHTML)
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
		Title:   data.Resource.Title,
		Content: content,
	}); err != nil {
		return "", fmt.Errorf("render default page template: %w", err)
	}

	return output.String(), nil
}

var _ core.ResourceTemplateRenderer = (*PageDefaultRenderer)(nil)
