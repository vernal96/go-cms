package seo

import (
	"errors"
	"strings"
	"testing"
)

func TestCompileTemplateRendersStrictVariables(t *testing.T) {
	t.Parallel()
	compiled, err := CompileTemplate(
		"{{ resource.title }} — {{ site.domain }}",
		map[string]struct{}{
			"resource.title": {},
			"site.domain":    {},
		},
		100,
	)
	if err != nil {
		t.Fatal(err)
	}
	value, missing := compiled.render(func(variable string) (string, bool) {
		return map[string]string{
			"resource.title": "Контакты",
			"site.domain":    "example.com",
		}[variable], true
	})
	if value != "Контакты — example.com" || len(missing) != 0 {
		t.Fatalf("render = %q, %#v", value, missing)
	}
}

func TestCompileTemplateRejectsUnknownInvalidAndLongSources(t *testing.T) {
	t.Parallel()
	allowed := map[string]struct{}{"resource.title": {}}
	for _, source := range []string{
		"{{ resource.content }}",
		"{{ resource.title | html }}",
		"{{ resource.title",
		"resource.title }}",
	} {
		if _, err := CompileTemplate(source, allowed, 100); err == nil {
			t.Fatalf("source %q was accepted", source)
		}
	}
	_, err := CompileTemplate(strings.Repeat("я", 11), allowed, 10)
	var templateError TemplateError
	if !errors.As(err, &templateError) ||
		!strings.Contains(templateError.Error(), "exceeds 10") {
		t.Fatalf("length error = %v", err)
	}
}
