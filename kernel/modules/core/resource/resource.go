package resource

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
)

type ID int64

var (
	ErrNotFound         = errors.New("resource not found")
	ErrConflict         = errors.New("resource conflict")
	ErrInvalidReference = errors.New("invalid resource reference")
	ErrInvalidTree      = errors.New("invalid resource tree")
	ErrReferenced       = errors.New("resource is referenced")
)

type Resource struct {
	ID               ID
	SiteID           site.ID
	ParentID         *ID
	Type             resourcetype.Code
	Template         *template.Code
	ContentType      *string
	Title            string
	MenuTitle        string
	Slug             string
	Path             *string
	Content          string
	TargetResourceID *ID
	ExternalURL      *string
	IsPublic         bool
	IsSearchable     bool
	InMenu           bool
	InSitemap        bool
	Sort             int
	PublishedAt      *time.Time
	UnpublishedAt    *time.Time
	Settings         map[string]any
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type CreateInput struct {
	SiteID           site.ID
	ParentID         *ID
	Type             resourcetype.Code
	Template         *template.Code
	ContentType      *string
	Title            string
	MenuTitle        string
	Slug             string
	Content          string
	TargetResourceID *ID
	ExternalURL      *string
	IsPublic         *bool
	IsSearchable     *bool
	InMenu           *bool
	InSitemap        *bool
	Sort             int
	PublishedAt      *time.Time
	UnpublishedAt    *time.Time
	Settings         map[string]any
}

type UpdateInput struct {
	ID               ID
	ParentID         *ID
	Type             resourcetype.Code
	Template         *template.Code
	ContentType      *string
	Title            string
	MenuTitle        string
	Slug             string
	Content          string
	TargetResourceID *ID
	ExternalURL      *string
	IsPublic         bool
	IsSearchable     bool
	InMenu           bool
	InSitemap        bool
	Sort             int
	PublishedAt      *time.Time
	UnpublishedAt    *time.Time
	Settings         map[string]any
}

type Node struct {
	Resource Resource
	Children []Node
}

type Repository interface {
	Create(context.Context, Resource) (Resource, error)
	ByID(context.Context, ID) (Resource, error)
	ByPath(context.Context, site.ID, string) (Resource, error)
	ListBySite(context.Context, site.ID) ([]Resource, error)
	Update(context.Context, Resource) (Resource, error)
	Delete(context.Context, ID) error
}

var slugPattern = regexp.MustCompile(
	`^[a-z0-9]+(?:-[a-z0-9]+)*$`,
)

func Clone(item Resource) Resource {
	item.ParentID = cloneID(item.ParentID)
	item.Template = cloneTemplateCode(item.Template)
	item.ContentType = cloneString(item.ContentType)
	item.Path = cloneString(item.Path)
	item.TargetResourceID = cloneID(item.TargetResourceID)
	item.ExternalURL = cloneString(item.ExternalURL)
	item.PublishedAt = cloneTime(item.PublishedAt)
	item.UnpublishedAt = cloneTime(item.UnpublishedAt)
	item.Settings = cloneMap(item.Settings)
	return item
}

func BuildPath(
	parent *Resource,
	slug string,
) (*string, error) {
	if parent == nil {
		path := "/"
		if slug != "" {
			path += slug
		}
		return &path, nil
	}

	if slug == "" {
		return nil, errors.New(
			"child resource slug is empty",
		)
	}
	if parent.Path == nil {
		return nil, errors.New(
			"route resource parent has no path",
		)
	}

	path := *parent.Path
	if path == "/" {
		path += slug
	} else {
		path += "/" + slug
	}
	return &path, nil
}

func validSlug(slug string, parentID *ID) bool {
	if slug == "" {
		return parentID == nil
	}

	return slugPattern.MatchString(slug)
}

func cloneID(value *ID) *ID {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneTemplateCode(value *template.Code) *template.Code {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	result := *value
	return &result
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := *value
	return &result
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
