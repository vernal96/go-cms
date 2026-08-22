package resource

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/security"
)

type LibraryItem struct {
	ID             ID
	SiteID         site.ID
	LibraryID      ID
	Template       *template.Code
	ContentType    *string
	Title          string
	Slug           string
	Annotation     string
	Content        string
	ImageMediaID   *media.ID
	IsPublic       bool
	IsSearchable   bool
	PublishedAt    *time.Time
	UnpublishedAt  *time.Time
	Fields         map[string]any
	FieldValues    []field.StoredValue
	Widgets        []widget.Binding
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CreatedBy      *security.UserID
	UpdatedBy      *security.UserID
	DeletedAt      *time.Time
	DeletedBy      *security.UserID
	FileReferences map[string]file.ID
}

type CreateLibraryItemInput struct {
	SiteID        site.ID
	LibraryID     ID
	Template      *template.Code
	ContentType   *string
	Title         string
	Slug          string
	Annotation    string
	Content       string
	ImageMediaID  *media.ID
	IsPublic      *bool
	IsSearchable  *bool
	PublishedAt   *time.Time
	UnpublishedAt *time.Time
	Fields        map[string]any
}

type UpdateLibraryItemInput struct {
	ID            ID
	Template      *template.Code
	ContentType   *string
	Title         string
	Slug          string
	Annotation    string
	Content       string
	ImageMediaID  *media.ID
	IsPublic      bool
	IsSearchable  bool
	PublishedAt   *time.Time
	UnpublishedAt *time.Time
	Fields        map[string]any
}

type LibraryItemQuery struct {
	SiteID     site.ID
	LibraryID  ID
	AfterID    ID
	Limit      int
	Search     string
	PublicOnly bool
	Deleted    *bool
}

type LibraryItemPage struct {
	Items      []LibraryItem
	NextCursor string
}

type LibraryItemRepository interface {
	CreateLibraryItem(context.Context, *security.UserID, LibraryItem) (LibraryItem, error)
	LibraryItemByID(context.Context, ID) (LibraryItem, error)
	UpdateLibraryItem(context.Context, *security.UserID, LibraryItem, LibraryItem) (LibraryItem, error)
	SoftDeleteLibraryItem(context.Context, *security.UserID, ID) error
	RestoreLibraryItem(context.Context, *security.UserID, ID) error
	DeleteLibraryItem(context.Context, ID) error
	MoveLibraryItem(context.Context, *security.UserID, ID, ID) (LibraryItem, error)
	QueryLibraryItems(context.Context, LibraryItemQuery) (LibraryItemPage, error)
	ResolveLibraryItemRoute(context.Context, site.ID, string) (LibraryItem, Resource, error)
}

type LibraryService struct {
	repository LibraryItemRepository
	common     *Service
}

func NewLibraryService(repository LibraryItemRepository, common *Service) (*LibraryService, error) {
	if repository == nil || common == nil {
		return nil, errors.New("library item dependencies are nil")
	}
	return &LibraryService{repository: repository, common: common}, nil
}

func (s *LibraryService) Create(ctx context.Context, actor security.Actor, input CreateLibraryItemInput) (LibraryItem, error) {
	if err := validateContext(ctx, "library item create"); err != nil {
		return LibraryItem{}, err
	}
	if err := s.common.authorizer.Check(ctx, actor, createPermission); err != nil {
		return LibraryItem{}, err
	}
	library, runtime, err := s.library(ctx, input.SiteID, input.LibraryID)
	if err != nil {
		return LibraryItem{}, err
	}
	templateCode := cloneTemplateCode(input.Template)
	if templateCode == nil {
		if value, ok := library.TypeSettings["default_item_template"].(string); ok && value != "" {
			code := template.Code(value)
			templateCode = &code
		}
	}
	item := LibraryItem{
		SiteID: input.SiteID, LibraryID: input.LibraryID, Template: templateCode,
		ContentType: cloneString(input.ContentType), Title: input.Title, Slug: input.Slug,
		Annotation: input.Annotation, Content: input.Content, ImageMediaID: cloneMediaID(input.ImageMediaID),
		IsPublic: boolDefault(input.IsPublic, true), IsSearchable: boolDefault(input.IsSearchable, true),
		PublishedAt: cloneTime(input.PublishedAt), UnpublishedAt: cloneTime(input.UnpublishedAt),
		Fields: cloneMap(input.Fields), CreatedBy: actor.AuditUserID(), UpdatedBy: actor.AuditUserID(),
	}
	if item.Slug == "" {
		item.Slug = GenerateSlug(item.Title)
	}
	item, err = s.normalize(ctx, actor, item, runtime, nil)
	if err != nil {
		return LibraryItem{}, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	created, err := s.repository.CreateLibraryItem(ctx, actor.AuditUserID(), item)
	if err != nil {
		return LibraryItem{}, fmt.Errorf("create library item: %w", err)
	}
	return cloneLibraryItem(created), nil
}

func (s *LibraryService) Get(ctx context.Context, actor security.Actor, id ID) (LibraryItem, error) {
	if err := validateContext(ctx, "library item get"); err != nil {
		return LibraryItem{}, err
	}
	if err := s.common.authorizer.Check(ctx, actor, readPermission); err != nil {
		return LibraryItem{}, err
	}
	item, err := s.repository.LibraryItemByID(ctx, id)
	if err != nil {
		return LibraryItem{}, fmt.Errorf("get library item %d: %w", id, err)
	}
	return cloneLibraryItem(item), nil
}

func (s *LibraryService) Update(ctx context.Context, actor security.Actor, input UpdateLibraryItemInput) (LibraryItem, error) {
	if err := validateContext(ctx, "library item update"); err != nil {
		return LibraryItem{}, err
	}
	if err := s.common.authorizer.Check(ctx, actor, updatePermission); err != nil {
		return LibraryItem{}, err
	}
	current, err := s.repository.LibraryItemByID(ctx, input.ID)
	if err != nil {
		return LibraryItem{}, err
	}
	_, runtime, err := s.library(ctx, current.SiteID, current.LibraryID)
	if err != nil {
		return LibraryItem{}, err
	}
	item := current
	item.Template = cloneTemplateCode(input.Template)
	item.ContentType = cloneString(input.ContentType)
	item.Title, item.Slug, item.Annotation, item.Content = input.Title, input.Slug, input.Annotation, input.Content
	item.ImageMediaID = cloneMediaID(input.ImageMediaID)
	item.IsPublic, item.IsSearchable = input.IsPublic, input.IsSearchable
	item.PublishedAt, item.UnpublishedAt = cloneTime(input.PublishedAt), cloneTime(input.UnpublishedAt)
	item.Fields = cloneMap(input.Fields)
	item.UpdatedBy = actor.AuditUserID()
	if item.Slug == "" {
		item.Slug = GenerateSlug(item.Title)
	}
	item, err = s.normalize(ctx, actor, item, runtime, current.FileReferences)
	if err != nil {
		return LibraryItem{}, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	updated, err := s.repository.UpdateLibraryItem(ctx, actor.AuditUserID(), current, item)
	if err != nil {
		return LibraryItem{}, fmt.Errorf("update library item: %w", err)
	}
	return cloneLibraryItem(updated), nil
}

func (s *LibraryService) Move(ctx context.Context, actor security.Actor, id, targetLibraryID ID) (LibraryItem, error) {
	if err := s.common.authorizer.Check(ctx, actor, updatePermission); err != nil {
		return LibraryItem{}, err
	}
	item, err := s.repository.LibraryItemByID(ctx, id)
	if err != nil {
		return LibraryItem{}, err
	}
	if _, _, err := s.library(ctx, item.SiteID, targetLibraryID); err != nil {
		return LibraryItem{}, err
	}
	moved, err := s.repository.MoveLibraryItem(ctx, actor.AuditUserID(), id, targetLibraryID)
	if err != nil {
		return LibraryItem{}, fmt.Errorf("move library item: %w", err)
	}
	return cloneLibraryItem(moved), nil
}

func (s *LibraryService) Query(ctx context.Context, actor security.Actor, query LibraryItemQuery) (LibraryItemPage, error) {
	if err := s.common.authorizer.Check(ctx, actor, readPermission); err != nil {
		return LibraryItemPage{}, err
	}
	if query.Limit <= 0 || query.Limit > 100 || query.SiteID <= 0 || query.LibraryID <= 0 {
		return LibraryItemPage{}, ErrInvalid
	}
	if _, _, err := s.library(ctx, query.SiteID, query.LibraryID); err != nil {
		return LibraryItemPage{}, err
	}
	page, err := s.repository.QueryLibraryItems(ctx, query)
	if err != nil {
		return LibraryItemPage{}, fmt.Errorf("query library items: %w", err)
	}
	for index := range page.Items {
		page.Items[index] = cloneLibraryItem(page.Items[index])
	}
	return page, nil
}

func (s *LibraryService) Delete(ctx context.Context, actor security.Actor, id ID, permanent bool) error {
	if err := s.common.authorizer.Check(ctx, actor, deletePermission); err != nil {
		return err
	}
	if permanent {
		return s.repository.DeleteLibraryItem(ctx, id)
	}
	return s.repository.SoftDeleteLibraryItem(ctx, actor.AuditUserID(), id)
}

func (s *LibraryService) Restore(ctx context.Context, actor security.Actor, id ID) error {
	if err := s.common.authorizer.Check(ctx, actor, deletePermission); err != nil {
		return err
	}
	return s.repository.RestoreLibraryItem(ctx, actor.AuditUserID(), id)
}

func (s *LibraryService) ResolvePublished(ctx context.Context, actor security.Actor, siteID site.ID, path string) (LibraryItem, Resource, error) {
	if err := s.common.authorizer.Check(ctx, actor, readPermission); err != nil {
		return LibraryItem{}, Resource{}, err
	}
	item, library, err := s.repository.ResolveLibraryItemRoute(ctx, siteID, path)
	if err != nil {
		return LibraryItem{}, Resource{}, err
	}
	now := time.Now().UTC()
	if library.DeletedAt != nil || item.DeletedAt != nil || !library.IsPublic || !item.IsPublic ||
		(item.PublishedAt != nil && now.Before(*item.PublishedAt)) || (item.UnpublishedAt != nil && !now.Before(*item.UnpublishedAt)) {
		return LibraryItem{}, Resource{}, ErrNotFound
	}
	return cloneLibraryItem(item), Clone(library), nil
}

func (s *LibraryService) library(ctx context.Context, siteID site.ID, id ID) (Resource, *site.Runtime, error) {
	item, err := s.common.repository.ByID(ctx, id)
	if err != nil {
		return Resource{}, nil, err
	}
	if item.SiteID != siteID || item.Type != resourcetype.Library || item.DeletedAt != nil {
		return Resource{}, nil, ErrInvalidReference
	}
	runtime, exists := s.common.sites.RuntimeByID(siteID)
	if !exists {
		return Resource{}, nil, fmt.Errorf("resource site %d not found", siteID)
	}
	return item, runtime, nil
}

func (s *LibraryService) normalize(ctx context.Context, actor security.Actor, item LibraryItem, runtime *site.Runtime, trusted map[string]file.ID) (LibraryItem, error) {
	item.Title = strings.TrimSpace(item.Title)
	if item.Title == "" || !validSlug(item.Slug, pointerID(item.LibraryID)) {
		return LibraryItem{}, errors.New("library item title or slug is invalid")
	}
	if item.PublishedAt != nil && item.UnpublishedAt != nil && !item.UnpublishedAt.After(*item.PublishedAt) {
		return LibraryItem{}, errors.New("library item publication window is invalid")
	}
	if item.ImageMediaID != nil {
		if err := s.common.validateImageMedia(ctx, *item.ImageMediaID); err != nil {
			return LibraryItem{}, err
		}
	}
	if item.ContentType == nil {
		value := "html"
		item.ContentType = &value
	}
	if *item.ContentType != "html" {
		return LibraryItem{}, errors.New("library item content_type is unsupported")
	}
	if item.Template == nil {
		if len(item.Fields) != 0 {
			return LibraryItem{}, errors.New("library item without template has fields")
		}
		item.Fields, item.FieldValues, item.FileReferences = map[string]any{}, nil, nil
		return item, nil
	}
	templateRuntime, exists := runtime.Profile().Template(*item.Template)
	if !exists {
		return LibraryItem{}, fmt.Errorf("library item references unknown template %q", *item.Template)
	}
	fields, err := templateRuntime.FieldSchema().Validate(item.Fields)
	if err != nil {
		return LibraryItem{}, err
	}
	item.FieldValues, err = templateRuntime.FieldSchema().StoredValues(fields)
	if err != nil {
		return LibraryItem{}, err
	}
	references, err := templateRuntime.FieldSchema().FileReferences(fields)
	if err != nil {
		return LibraryItem{}, err
	}
	if err := s.common.validateFileReferences(ctx, actor, references, trusted); err != nil {
		return LibraryItem{}, err
	}
	item.Fields, item.FileReferences = fields, resourceFileReferenceMap(references)
	return item, nil
}

func EffectiveLibraryItemURL(library Resource, item LibraryItem) (string, error) {
	if library.Path == nil {
		return "", errors.New("library has no path")
	}
	pattern, _ := library.TypeSettings["item_url_pattern"].(string)
	if pattern == "" {
		pattern = resourcetype.DefaultItemURLPattern
	}
	if err := resourcetype.ValidateItemURLPattern(pattern); err != nil {
		return "", err
	}
	date := item.CreatedAt.UTC()
	if item.PublishedAt != nil {
		date = item.PublishedAt.UTC()
	}
	replacements := map[string]string{
		"{id}": strconv.FormatInt(int64(item.ID), 10), "{slug}": item.Slug,
		"{year}": fmt.Sprintf("%04d", date.Year()), "{month}": fmt.Sprintf("%02d", date.Month()), "{day}": fmt.Sprintf("%02d", date.Day()),
	}
	result := pattern
	for token, value := range replacements {
		result = strings.ReplaceAll(result, token, value)
	}
	return strings.TrimRight(*library.Path, "/") + result, nil
}

type LibraryRouteKey struct {
	ID   ID
	Slug string
}

func MatchLibraryItemPattern(pattern, relativePath string) (LibraryRouteKey, bool) {
	if err := resourcetype.ValidateItemURLPattern(pattern); err != nil {
		return LibraryRouteKey{}, false
	}
	var expression strings.Builder
	expression.WriteByte('^')
	for cursor := 0; cursor < len(pattern); {
		open := strings.IndexByte(pattern[cursor:], '{')
		if open < 0 {
			expression.WriteString(regexp.QuoteMeta(pattern[cursor:]))
			break
		}
		open += cursor
		expression.WriteString(regexp.QuoteMeta(pattern[cursor:open]))
		close := strings.IndexByte(pattern[open:], '}') + open
		switch pattern[open+1 : close] {
		case "id":
			expression.WriteString(`(?P<id>[1-9][0-9]*)`)
		case "slug":
			expression.WriteString(`(?P<slug>[a-z0-9]+(?:-[a-z0-9]+)*)`)
		case "year":
			expression.WriteString(`[0-9]{4}`)
		case "month", "day":
			expression.WriteString(`[0-9]{2}`)
		}
		cursor = close + 1
	}
	expression.WriteByte('$')
	compiled := regexp.MustCompile(expression.String())
	match := compiled.FindStringSubmatch(relativePath)
	if match == nil {
		return LibraryRouteKey{}, false
	}
	result := LibraryRouteKey{}
	for index, name := range compiled.SubexpNames() {
		if index == 0 || name == "" {
			continue
		}
		if name == "slug" {
			result.Slug = match[index]
		}
		if name == "id" {
			parsed, _ := strconv.ParseInt(match[index], 10, 64)
			result.ID = ID(parsed)
		}
	}
	return result, true
}

func EncodeLibraryCursor(id ID) string {
	if id <= 0 {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(int64(id), 10)))
}

func DecodeLibraryCursor(value string) (ID, error) {
	if value == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return 0, ErrInvalid
	}
	id, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil || id <= 0 {
		return 0, ErrInvalid
	}
	return ID(id), nil
}

func pointerID(id ID) *ID { return &id }

func cloneLibraryItem(item LibraryItem) LibraryItem {
	item.Template = cloneTemplateCode(item.Template)
	item.ContentType = cloneString(item.ContentType)
	item.ImageMediaID = cloneMediaID(item.ImageMediaID)
	item.PublishedAt = cloneTime(item.PublishedAt)
	item.UnpublishedAt = cloneTime(item.UnpublishedAt)
	item.Fields = cloneMap(item.Fields)
	item.FieldValues = cloneStoredValues(item.FieldValues)
	item.Widgets = widget.CloneBindings(item.Widgets)
	item.FileReferences = cloneFileReferences(item.FileReferences)
	item.CreatedBy = cloneUserID(item.CreatedBy)
	item.UpdatedBy = cloneUserID(item.UpdatedBy)
	item.DeletedAt = cloneTime(item.DeletedAt)
	item.DeletedBy = cloneUserID(item.DeletedBy)
	return item
}
