package seo

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var variablePattern = regexp.MustCompile(
	`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`,
)

type templatePart struct {
	literal  string
	variable string
}

type Template struct {
	parts []templatePart
}

type TemplateError struct {
	Variable string
	Message  string
}

func (e TemplateError) Error() string {
	if e.Variable == "" {
		return e.Message
	}
	return fmt.Sprintf("variable %q: %s", e.Variable, e.Message)
}

func CompileTemplate(
	source string,
	allowed map[string]struct{},
	maxLength int,
) (Template, error) {
	if maxLength <= 0 {
		return Template{}, TemplateError{Message: "maximum template length is invalid"}
	}
	if utf8.RuneCountInString(source) > maxLength {
		return Template{}, TemplateError{Message: fmt.Sprintf(
			"template exceeds %d characters",
			maxLength,
		)}
	}

	parts := make([]templatePart, 0, 4)
	rest := source
	for rest != "" {
		opening := strings.Index(rest, "{{")
		closing := strings.Index(rest, "}}")
		if closing >= 0 && (opening < 0 || closing < opening) {
			return Template{}, TemplateError{Message: "unexpected closing delimiter"}
		}
		if opening < 0 {
			parts = append(parts, templatePart{literal: rest})
			break
		}
		if opening > 0 {
			parts = append(parts, templatePart{literal: rest[:opening]})
		}
		rest = rest[opening+2:]
		closing = strings.Index(rest, "}}")
		if closing < 0 {
			return Template{}, TemplateError{Message: "unclosed variable"}
		}
		variable := strings.TrimSpace(rest[:closing])
		if !variablePattern.MatchString(variable) {
			return Template{}, TemplateError{
				Variable: variable,
				Message:  "invalid variable syntax",
			}
		}
		if _, exists := allowed[variable]; !exists {
			return Template{}, TemplateError{
				Variable: variable,
				Message:  "unknown variable",
			}
		}
		parts = append(parts, templatePart{variable: variable})
		rest = rest[closing+2:]
	}
	return Template{parts: parts}, nil
}

func (t Template) render(resolve func(string) (string, bool)) (string, []string) {
	var result strings.Builder
	missing := make([]string, 0)
	for _, part := range t.parts {
		if part.variable == "" {
			result.WriteString(part.literal)
			continue
		}
		value, exists := resolve(part.variable)
		if !exists || strings.TrimSpace(value) == "" {
			missing = append(missing, part.variable)
			continue
		}
		result.WriteString(value)
	}
	return result.String(), missing
}
