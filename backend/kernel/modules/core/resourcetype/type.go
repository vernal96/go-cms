package resourcetype

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/vernal96/go-cms/kernel/modules/core/template"
)

type Code string

const (
	Page         Code = "page"
	Link         Code = "link"
	ResourceLink Code = "resource_link"
)

type PathMode string

const (
	PathRoute PathMode = "route"
	PathNone  PathMode = "none"
)

type Payload struct {
	Template         *template.Code
	ContentType      *string
	Content          string
	TargetResourceID *int64
	ExternalURL      *string
	Settings         map[string]any
}

type Type interface {
	Code() Code
	PathMode() PathMode
	Normalize(Payload) (Payload, error)
}

func StandardTypes() []Type {
	return []Type{
		pageType{},
		linkType{},
		resourceLinkType{},
	}
}

type pageType struct{}

func (pageType) Code() Code {
	return Page
}

func (pageType) PathMode() PathMode {
	return PathRoute
}

func (pageType) Normalize(payload Payload) (Payload, error) {
	if payload.Template == nil || *payload.Template == "" {
		return Payload{}, errors.New("page template is required")
	}
	if payload.TargetResourceID != nil {
		return Payload{}, errors.New(
			"page target_resource_id must be empty",
		)
	}
	if payload.ExternalURL != nil {
		return Payload{}, errors.New("page external_url must be empty")
	}

	if payload.ContentType == nil || *payload.ContentType == "" {
		contentType := "html"
		payload.ContentType = &contentType
	}
	if !contentTypePattern.MatchString(*payload.ContentType) {
		return Payload{}, fmt.Errorf(
			"invalid page content_type %q",
			*payload.ContentType,
		)
	}

	return clonePayload(payload), nil
}

type linkType struct{}

func (linkType) Code() Code {
	return Link
}

func (linkType) PathMode() PathMode {
	return PathRoute
}

func (linkType) Normalize(payload Payload) (Payload, error) {
	if payload.Template != nil {
		return Payload{}, errors.New("link template must be empty")
	}
	if payload.ContentType != nil {
		return Payload{}, errors.New("link content_type must be empty")
	}
	if payload.Content != "" {
		return Payload{}, errors.New("link content must be empty")
	}
	if payload.TargetResourceID != nil {
		return Payload{}, errors.New(
			"link target_resource_id must be empty",
		)
	}
	if len(payload.Settings) != 0 {
		return Payload{}, errors.New("link settings must be empty")
	}
	if payload.ExternalURL == nil ||
		!validExternalURL(*payload.ExternalURL) {
		return Payload{}, errors.New(
			"link external_url is invalid",
		)
	}

	return clonePayload(payload), nil
}

type resourceLinkType struct{}

func (resourceLinkType) Code() Code {
	return ResourceLink
}

func (resourceLinkType) PathMode() PathMode {
	return PathRoute
}

func (resourceLinkType) Normalize(
	payload Payload,
) (Payload, error) {
	if payload.Template != nil {
		return Payload{}, errors.New(
			"resource_link template must be empty",
		)
	}
	if payload.ContentType != nil {
		return Payload{}, errors.New(
			"resource_link content_type must be empty",
		)
	}
	if payload.Content != "" {
		return Payload{}, errors.New(
			"resource_link content must be empty",
		)
	}
	if payload.ExternalURL != nil {
		return Payload{}, errors.New(
			"resource_link external_url must be empty",
		)
	}
	if len(payload.Settings) != 0 {
		return Payload{}, errors.New(
			"resource_link settings must be empty",
		)
	}
	if payload.TargetResourceID == nil ||
		*payload.TargetResourceID <= 0 {
		return Payload{}, errors.New(
			"resource_link target_resource_id is required",
		)
	}

	return clonePayload(payload), nil
}

var contentTypePattern = regexp.MustCompile(
	`^[a-z0-9][a-z0-9.+-]*$`,
)

func validExternalURL(value string) bool {
	if value == "" || strings.TrimSpace(value) != value {
		return false
	}

	if strings.HasPrefix(value, "/") {
		if strings.HasPrefix(value, "//") {
			return false
		}
		parsed, err := url.ParseRequestURI(value)
		return err == nil && parsed.Scheme == "" && parsed.Host == ""
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return false
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return parsed.Host != ""

	case "mailto", "tel":
		return parsed.Opaque != "" || parsed.Path != ""

	default:
		return false
	}
}

func clonePayload(payload Payload) Payload {
	if payload.Template != nil {
		value := *payload.Template
		payload.Template = &value
	}
	if payload.ContentType != nil {
		value := *payload.ContentType
		payload.ContentType = &value
	}
	if payload.TargetResourceID != nil {
		value := *payload.TargetResourceID
		payload.TargetResourceID = &value
	}
	if payload.ExternalURL != nil {
		value := *payload.ExternalURL
		payload.ExternalURL = &value
	}

	payload.Settings = cloneMap(payload.Settings)
	return payload
}

func cloneMap(source map[string]any) map[string]any {
	if source == nil {
		return map[string]any{}
	}

	result := make(map[string]any, len(source))
	for key, value := range source {
		result[key] = cloneValue(value)
	}
	return result
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		result := make([]any, len(typed))
		for index, item := range typed {
			result[index] = cloneValue(item)
		}
		return result
	case []string:
		return append([]string(nil), typed...)
	default:
		return typed
	}
}
