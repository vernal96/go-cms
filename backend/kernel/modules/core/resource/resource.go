package resource

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/security"
)

type ID int64

const ImageMediaUsage media.UsageKind = "resource.image"

var (
	ErrNotFound         = errors.New("resource not found")
	ErrConflict         = errors.New("resource conflict")
	ErrInvalid          = errors.New("invalid resource")
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
	Annotation       string
	Content          string
	ImageMediaID     *media.ID
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
	Widgets          []widget.Binding
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CreatedBy        *security.UserID
	UpdatedBy        *security.UserID
	DeletedAt        *time.Time
	DeletedBy        *security.UserID
	FileReferences   map[string]file.ID
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
	Annotation       string
	Content          string
	ImageMediaID     *media.ID
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
	Annotation       string
	Content          string
	ImageMediaID     *media.ID
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

type CreateWidgetInput struct {
	Code         widget.Code
	Area         widget.AreaCode
	View         widget.ViewCode
	Columns      int
	MarginTop    int
	MarginBottom int
	Enabled      *bool
	Params       map[string]any
}

type UpdateWidgetInput struct {
	View         widget.ViewCode
	Columns      int
	MarginTop    int
	MarginBottom int
	Enabled      *bool
	Params       map[string]any
}

type Node struct {
	Resource Resource
	Children []Node
}

type Child struct {
	ID          ID
	SiteID      site.ID
	ParentID    *ID
	Type        resourcetype.Code
	Template    *template.Code
	Title       string
	MenuTitle   string
	Sort        int
	DeletedAt   *time.Time
	HasChildren bool
}

type ValidateImageMedia func(context.Context, media.ID) error

type Repository interface {
	Create(
		context.Context,
		*security.UserID,
		Resource,
		ValidateImageMedia,
	) (Resource, error)
	ByID(context.Context, ID) (Resource, error)
	ByPath(context.Context, site.ID, string) (Resource, error)
	ListBySite(context.Context, site.ID) ([]Resource, error)
	Update(
		context.Context,
		*security.UserID,
		Resource,
		Resource,
		ValidateImageMedia,
	) (Resource, error)
	Delete(context.Context, ID) error
}

type WidgetRepository interface {
	CreateWidget(context.Context, ID, widget.Binding) (widget.Binding, error)
	UpdateWidget(context.Context, ID, widget.Binding) (widget.Binding, error)
	DeleteWidget(context.Context, ID, widget.BindingID) error
	ReorderWidgets(context.Context, ID, []widget.Order) ([]widget.Binding, error)
}

type LifecycleRepository interface {
	Repository
	SoftDelete(context.Context, *security.UserID, ID) error
	Restore(context.Context, *security.UserID, ID, bool) error
}

type ManagementRepository interface {
	Repository
	ExistsInSite(context.Context, site.ID, ID) (bool, error)
	ListChildren(context.Context, site.ID, *ID) ([]Child, error)
}

type StatisticsRepository interface {
	Statistics(context.Context, StatisticsQuery) (Statistics, error)
}

type StatisticsQuery struct {
	Scope   site.Scope
	SiteIDs []site.ID
}

type Statistics struct {
	Total  int
	BySite map[site.ID]int
}

var slugPattern = regexp.MustCompile(
	`^[a-z0-9]+(?:-[a-z0-9]+)*$`,
)

func Clone(item Resource) Resource {
	item.ParentID = cloneID(item.ParentID)
	item.Template = cloneTemplateCode(item.Template)
	item.ContentType = cloneString(item.ContentType)
	item.Path = cloneString(item.Path)
	item.ImageMediaID = cloneMediaID(item.ImageMediaID)
	item.TargetResourceID = cloneID(item.TargetResourceID)
	item.ExternalURL = cloneString(item.ExternalURL)
	item.PublishedAt = cloneTime(item.PublishedAt)
	item.UnpublishedAt = cloneTime(item.UnpublishedAt)
	item.Settings = cloneMap(item.Settings)
	item.Widgets = widget.CloneBindings(item.Widgets)
	item.CreatedBy = cloneUserID(item.CreatedBy)
	item.UpdatedBy = cloneUserID(item.UpdatedBy)
	item.DeletedAt = cloneTime(item.DeletedAt)
	item.DeletedBy = cloneUserID(item.DeletedBy)
	item.FileReferences = cloneFileReferences(item.FileReferences)
	return item
}

func cloneFileReferences(source map[string]file.ID) map[string]file.ID {
	if source == nil {
		return nil
	}
	result := make(map[string]file.ID, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func cloneUserID(value *security.UserID) *security.UserID {
	if value == nil {
		return nil
	}
	result := *value
	return &result
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

func cloneMediaID(value *media.ID) *media.ID {
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
