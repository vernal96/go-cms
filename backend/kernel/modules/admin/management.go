package admin

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

var ErrValidation = errors.New("admin validation failed")

const (
	SiteReadPermission       permission.Code = "core.site.read"
	SiteCreatePermission     permission.Code = "core.site.create"
	SiteUpdatePermission     permission.Code = "core.site.update"
	SiteDeletePermission     permission.Code = "core.site.delete"
	ResourceReadPermission   permission.Code = "core.resource.read"
	ResourceCreatePermission permission.Code = "core.resource.create"
	ResourceUpdatePermission permission.Code = "core.resource.update"
	ResourceDeletePermission permission.Code = "core.resource.delete"
)

var AdminPermissionCodes = []permission.Code{
	AccessPermission,
	SiteReadPermission,
	SiteCreatePermission,
	SiteUpdatePermission,
	SiteDeletePermission,
	ResourceReadPermission,
	ResourceCreatePermission,
	ResourceUpdatePermission,
	ResourceDeletePermission,
}

type SiteAccessScope struct {
	All     bool
	SiteIDs []site.ID
}

type SiteAccessPolicy interface {
	Scope(context.Context, security.Actor) (SiteAccessScope, error)
	Check(context.Context, security.Actor, site.ID) error
}

type AllowAllSitesPolicy struct{}

func (AllowAllSitesPolicy) Scope(
	context.Context,
	security.Actor,
) (SiteAccessScope, error) {
	return SiteAccessScope{All: true}, nil
}

func (AllowAllSitesPolicy) Check(
	context.Context,
	security.Actor,
	site.ID,
) error {
	return nil
}

type ProfileResolver interface {
	ProfileRuntime(kernel.ProfileCode) (*kernel.ProfileRuntime, bool)
}

type Management struct {
	profiles      []kernel.Profile
	profileSource ProfileResolver
	repository    site.ManagementRepository
	sites         *site.Catalog
	resources     *resource.Service
	resourceRepo  resource.ManagementRepository
	authorizer    security.Authorizer
	policy        SiteAccessPolicy
}

type ManagementDependencies struct {
	Profiles           []kernel.Profile
	ProfileSource      ProfileResolver
	SiteRepository     site.ManagementRepository
	Sites              *site.Catalog
	Resources          *resource.Service
	ResourceRepository resource.ManagementRepository
	Authorizer         security.Authorizer
	SiteAccessPolicy   SiteAccessPolicy
}

func NewManagement(dependencies ManagementDependencies) (*Management, error) {
	if dependencies.ProfileSource == nil {
		return nil, errors.New("admin profile source is nil")
	}
	if dependencies.SiteRepository == nil || dependencies.Sites == nil {
		return nil, errors.New("admin site dependencies are nil")
	}
	if dependencies.Resources == nil || dependencies.ResourceRepository == nil {
		return nil, errors.New("admin resource dependencies are nil")
	}
	if dependencies.Authorizer == nil {
		return nil, errors.New("admin authorizer is nil")
	}
	if dependencies.SiteAccessPolicy == nil {
		return nil, errors.New("admin site access policy is nil")
	}
	profiles := make([]kernel.Profile, len(dependencies.Profiles))
	copy(profiles, dependencies.Profiles)
	return &Management{
		profiles:      profiles,
		profileSource: dependencies.ProfileSource,
		repository:    dependencies.SiteRepository,
		sites:         dependencies.Sites,
		resources:     dependencies.Resources,
		resourceRepo:  dependencies.ResourceRepository,
		authorizer:    dependencies.Authorizer,
		policy:        dependencies.SiteAccessPolicy,
	}, nil
}

type Pagination struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

type PermissionSet struct {
	Read   bool `json:"read"`
	Create bool `json:"create"`
	Update bool `json:"update"`
	Delete bool `json:"delete"`
}

type SiteDTO struct {
	ID          site.ID            `json:"id"`
	ProfileCode kernel.ProfileCode `json:"profile_code"`
	Domain      string             `json:"domain"`
	Locale      string             `json:"locale"`
	Settings    map[string]any     `json:"settings"`
	IsPublic    bool               `json:"is_public"`
}

type SiteOption struct {
	ID     site.ID `json:"id"`
	Domain string  `json:"domain"`
}

type SiteList struct {
	Items       []SiteDTO     `json:"items"`
	Pagination  Pagination    `json:"pagination"`
	Permissions PermissionSet `json:"permissions"`
}

type SiteOptions struct {
	Items      []SiteOption `json:"items"`
	Pagination Pagination   `json:"pagination"`
}

type SiteDetails struct {
	Site        SiteDTO       `json:"site"`
	Permissions PermissionSet `json:"permissions"`
}

type SiteProfile struct {
	Code      kernel.ProfileCode `json:"code"`
	Name      string             `json:"name"`
	Creatable bool               `json:"creatable"`
}

type SiteProfiles struct {
	Items []SiteProfile `json:"items"`
}

type SiteCreateInput struct {
	ProfileCode kernel.ProfileCode
	Domain      string
	Locale      string
	Settings    map[string]any
	IsPublic    bool
}

type SiteUpdateInput struct {
	ProfileCode kernel.ProfileCode
	Domain      string
	Locale      string
	IsPublic    bool
}

func (m *Management) ListSites(
	ctx context.Context,
	actor security.Actor,
	search string,
	page int,
	perPage int,
) (SiteList, error) {
	if err := m.authorizer.Check(ctx, actor, SiteReadPermission); err != nil {
		return SiteList{}, err
	}
	page, perPage, err := normalizePagination(page, perPage)
	if err != nil {
		return SiteList{}, err
	}
	scope, err := m.policy.Scope(ctx, actor)
	if err != nil {
		return SiteList{}, err
	}
	result, err := m.repository.ListPage(ctx, site.ListQuery{
		Search:  search,
		Page:    page,
		PerPage: perPage,
		Scope:   site.Scope{All: scope.All, SiteIDs: append([]site.ID(nil), scope.SiteIDs...)},
	})
	if err != nil {
		return SiteList{}, fmt.Errorf("list admin sites: %w", err)
	}
	items := make([]SiteDTO, len(result.Items))
	for index, item := range result.Items {
		items[index] = siteDTO(item)
	}
	permissions, err := m.sitePermissions(ctx, actor)
	if err != nil {
		return SiteList{}, err
	}
	return SiteList{
		Items:       items,
		Pagination:  Pagination{Page: page, PerPage: perPage, Total: result.Total},
		Permissions: permissions,
	}, nil
}

func (m *Management) ListSiteOptions(
	ctx context.Context,
	actor security.Actor,
	search string,
	page int,
	perPage int,
) (SiteOptions, error) {
	result, err := m.ListSites(ctx, actor, search, page, perPage)
	if err != nil {
		return SiteOptions{}, err
	}
	items := make([]SiteOption, len(result.Items))
	for index, item := range result.Items {
		items[index] = SiteOption{ID: item.ID, Domain: item.Domain}
	}
	return SiteOptions{Items: items, Pagination: result.Pagination}, nil
}

func (m *Management) Site(
	ctx context.Context,
	actor security.Actor,
	id site.ID,
) (SiteDetails, error) {
	if err := m.requireSite(ctx, actor, id, SiteReadPermission); err != nil {
		return SiteDetails{}, err
	}
	runtime, exists := m.sites.RuntimeByID(id)
	if !exists {
		return SiteDetails{}, site.ErrNotFound
	}
	permissions, err := m.sitePermissions(ctx, actor)
	if err != nil {
		return SiteDetails{}, err
	}
	return SiteDetails{Site: siteDTO(runtime.Site()), Permissions: permissions}, nil
}

func (m *Management) CreateSite(
	ctx context.Context,
	actor security.Actor,
	input SiteCreateInput,
) (SiteDetails, error) {
	if err := m.authorizer.Check(ctx, actor, SiteCreatePermission); err != nil {
		return SiteDetails{}, err
	}
	runtime, err := m.sites.Create(ctx, actor, site.CreateInput{
		ProfileCode: input.ProfileCode,
		Domain:      input.Domain,
		Locale:      input.Locale,
		Settings:    input.Settings,
		IsPublic:    input.IsPublic,
	})
	if err != nil {
		return SiteDetails{}, validationError(err)
	}
	permissions, err := m.sitePermissions(ctx, actor)
	if err != nil {
		return SiteDetails{}, err
	}
	return SiteDetails{Site: siteDTO(runtime.Site()), Permissions: permissions}, nil
}

func (m *Management) UpdateSite(
	ctx context.Context,
	actor security.Actor,
	id site.ID,
	input SiteUpdateInput,
) (SiteDetails, error) {
	if err := m.requireSite(ctx, actor, id, SiteUpdatePermission); err != nil {
		return SiteDetails{}, err
	}
	current, exists := m.sites.RuntimeByID(id)
	if !exists {
		return SiteDetails{}, site.ErrNotFound
	}
	runtime, err := m.sites.Update(ctx, actor, site.UpdateInput{
		ID:          id,
		ProfileCode: input.ProfileCode,
		Domain:      input.Domain,
		Locale:      input.Locale,
		Settings:    current.Site().Settings,
		IsPublic:    input.IsPublic,
	})
	if err != nil {
		return SiteDetails{}, validationError(err)
	}
	permissions, err := m.sitePermissions(ctx, actor)
	if err != nil {
		return SiteDetails{}, err
	}
	return SiteDetails{Site: siteDTO(runtime.Site()), Permissions: permissions}, nil
}

func (m *Management) DeleteSite(
	ctx context.Context,
	actor security.Actor,
	id site.ID,
) error {
	if err := m.requireSite(ctx, actor, id, SiteDeletePermission); err != nil {
		return err
	}
	return m.sites.Delete(ctx, actor, id)
}

func (m *Management) Profiles(
	ctx context.Context,
	actor security.Actor,
) (SiteProfiles, error) {
	if err := m.authorizer.Check(ctx, actor, SiteReadPermission); err != nil {
		return SiteProfiles{}, err
	}
	items := make([]SiteProfile, 0, len(m.profiles))
	for _, profile := range m.profiles {
		runtime, exists := m.profileSource.ProfileRuntime(profile.Code)
		if !exists {
			continue
		}
		_, err := runtime.ParamSchema().Validate(map[string]any{})
		items = append(items, SiteProfile{
			Code:      profile.Code,
			Name:      profile.Name,
			Creatable: err == nil,
		})
	}
	return SiteProfiles{Items: items}, nil
}

type ResourceTreeItem struct {
	ID             resource.ID    `json:"id"`
	ParentID       *resource.ID   `json:"parent_id"`
	TemplateCode   *template.Code `json:"template_code"`
	Icon           string         `json:"icon"`
	Title          string         `json:"title"`
	MenuTitle      string         `json:"menu_title"`
	DisplayTitle   string         `json:"display_title"`
	HasChildren    bool           `json:"has_children"`
	CanCreateChild bool           `json:"can_create_child"`
}

type ResourceChildren struct {
	Items       []ResourceTreeItem `json:"items"`
	Permissions struct {
		CreateRoot bool `json:"create_root"`
	} `json:"permissions"`
}

func (m *Management) ResourceChildren(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	parentID *resource.ID,
) (ResourceChildren, error) {
	if err := m.requireSite(ctx, actor, siteID, ResourceReadPermission); err != nil {
		return ResourceChildren{}, err
	}
	runtime, exists := m.sites.RuntimeByID(siteID)
	if !exists {
		return ResourceChildren{}, site.ErrNotFound
	}
	children, err := m.resourceRepo.ListChildren(ctx, siteID, parentID)
	if err != nil {
		return ResourceChildren{}, fmt.Errorf("list admin resource children: %w", err)
	}
	canCreate, err := m.allowed(ctx, actor, ResourceCreatePermission)
	if err != nil {
		return ResourceChildren{}, err
	}
	result := ResourceChildren{Items: make([]ResourceTreeItem, len(children))}
	result.Permissions.CreateRoot = canCreate
	for index, child := range children {
		result.Items[index] = treeItem(runtime, child, canCreate)
	}
	return result, nil
}

type ResourceTemplate struct {
	Code  template.Code `json:"code"`
	Label string        `json:"label"`
	Icon  string        `json:"icon"`
}

type ResourceType struct {
	Code  resourcetype.Code `json:"code"`
	Label string            `json:"label"`
}

type ResourceMetadata struct {
	Types     []ResourceType     `json:"types"`
	Templates []ResourceTemplate `json:"templates"`
}

func (m *Management) ResourceMetadata(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
) (ResourceMetadata, error) {
	if err := m.requireSite(ctx, actor, siteID, ResourceCreatePermission); err != nil {
		return ResourceMetadata{}, err
	}
	runtime, exists := m.sites.RuntimeByID(siteID)
	if !exists {
		return ResourceMetadata{}, site.ErrNotFound
	}
	definitions := runtime.Profile().Templates()
	templates := make([]ResourceTemplate, len(definitions))
	for index, definition := range definitions {
		templates[index] = ResourceTemplate{
			Code:  definition.Code,
			Label: definition.Label,
			Icon:  iconOrDefault(definition.Icon),
		}
	}
	types := []ResourceType{{Code: resourcetype.Link, Label: "Ссылка"}}
	if len(templates) > 0 {
		types = append([]ResourceType{{Code: resourcetype.Page, Label: "Страница"}}, types...)
	}
	return ResourceMetadata{Types: types, Templates: templates}, nil
}

type ResourceCreateInput struct {
	ParentID    *resource.ID
	Type        resourcetype.Code
	Template    *template.Code
	Title       string
	MenuTitle   string
	Slug        string
	ExternalURL *string
}

func (m *Management) CreateResource(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	input ResourceCreateInput,
) (ResourceTreeItem, error) {
	if err := m.requireSite(ctx, actor, siteID, ResourceCreatePermission); err != nil {
		return ResourceTreeItem{}, err
	}
	if input.Type != resourcetype.Page && input.Type != resourcetype.Link {
		return ResourceTreeItem{}, fmt.Errorf("%w: unsupported resource type", ErrValidation)
	}
	if input.ParentID != nil {
		exists, err := m.resourceRepo.ExistsInSite(ctx, siteID, *input.ParentID)
		if err != nil {
			return ResourceTreeItem{}, err
		}
		if !exists {
			return ResourceTreeItem{}, resource.ErrNotFound
		}
	}
	runtime, exists := m.sites.RuntimeByID(siteID)
	if !exists {
		return ResourceTreeItem{}, site.ErrNotFound
	}
	created, err := m.resources.Create(ctx, actor, resource.CreateInput{
		SiteID:      siteID,
		ParentID:    input.ParentID,
		Type:        input.Type,
		Template:    input.Template,
		Title:       input.Title,
		MenuTitle:   input.MenuTitle,
		Slug:        input.Slug,
		ExternalURL: input.ExternalURL,
		Settings:    map[string]any{},
	})
	if err != nil {
		if errors.Is(err, resource.ErrNotFound) {
			return ResourceTreeItem{}, err
		}
		return ResourceTreeItem{}, validationError(err)
	}
	return treeItem(runtime, resource.Child{
		ID:        created.ID,
		SiteID:    created.SiteID,
		ParentID:  created.ParentID,
		Type:      created.Type,
		Template:  created.Template,
		Title:     created.Title,
		MenuTitle: created.MenuTitle,
	}, true), nil
}

func (m *Management) requireSite(
	ctx context.Context,
	actor security.Actor,
	id site.ID,
	code permission.Code,
) error {
	if id <= 0 {
		return fmt.Errorf("%w: invalid site id", ErrValidation)
	}
	if err := m.authorizer.Check(ctx, actor, code); err != nil {
		return err
	}
	return m.policy.Check(ctx, actor, id)
}

func (m *Management) allowed(
	ctx context.Context,
	actor security.Actor,
	code permission.Code,
) (bool, error) {
	err := m.authorizer.Check(ctx, actor, code)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, security.ErrForbidden) || errors.Is(err, security.ErrUnauthenticated) {
		return false, nil
	}
	return false, err
}

func (m *Management) sitePermissions(
	ctx context.Context,
	actor security.Actor,
) (PermissionSet, error) {
	codes := []permission.Code{SiteReadPermission, SiteCreatePermission, SiteUpdatePermission, SiteDeletePermission}
	values := make([]bool, len(codes))
	for index, code := range codes {
		allowed, err := m.allowed(ctx, actor, code)
		if err != nil {
			return PermissionSet{}, err
		}
		values[index] = allowed
	}
	return PermissionSet{Read: values[0], Create: values[1], Update: values[2], Delete: values[3]}, nil
}

func normalizePagination(page, perPage int) (int, int, error) {
	if page == 0 {
		page = 1
	}
	if perPage == 0 {
		perPage = 10
	}
	if page < 1 || perPage < 1 || perPage > 100 {
		return 0, 0, fmt.Errorf("%w: invalid pagination", ErrValidation)
	}
	return page, perPage, nil
}

func siteDTO(item site.Site) SiteDTO {
	settings := make(map[string]any, len(item.Settings))
	for key, value := range item.Settings {
		settings[key] = value
	}
	return SiteDTO{
		ID:          item.ID,
		ProfileCode: item.ProfileCode,
		Domain:      item.Domain,
		Locale:      item.Locale,
		Settings:    settings,
		IsPublic:    item.IsPublic,
	}
}

func treeItem(runtime *site.Runtime, item resource.Child, canCreate bool) ResourceTreeItem {
	displayTitle := strings.TrimSpace(item.MenuTitle)
	if displayTitle == "" {
		displayTitle = item.Title
	}
	icon := "document"
	if item.Type == resourcetype.Link {
		icon = "link"
	} else if item.Template != nil {
		if templateRuntime, exists := runtime.Profile().Template(*item.Template); exists {
			icon = iconOrDefault(templateRuntime.Definition().Icon)
		}
	}
	return ResourceTreeItem{
		ID:             item.ID,
		ParentID:       item.ParentID,
		TemplateCode:   item.Template,
		Icon:           icon,
		Title:          item.Title,
		MenuTitle:      item.MenuTitle,
		DisplayTitle:   displayTitle,
		HasChildren:    item.HasChildren,
		CanCreateChild: canCreate,
	}
}

func iconOrDefault(icon string) string {
	icon = strings.TrimSpace(icon)
	switch icon {
	case "document", "link", "folder", "tickets":
		return icon
	default:
		return "document"
	}
}

func validationError(err error) error {
	if errors.Is(err, site.ErrConflict) || errors.Is(err, site.ErrNotFound) ||
		errors.Is(err, resource.ErrConflict) || errors.Is(err, resource.ErrNotFound) {
		return err
	}
	if errors.Is(err, site.ErrInvalid) || errors.Is(err, resource.ErrInvalid) ||
		errors.Is(err, resource.ErrInvalidReference) || errors.Is(err, resource.ErrInvalidTree) {
		return fmt.Errorf("%w: request data is invalid", ErrValidation)
	}
	return err
}
