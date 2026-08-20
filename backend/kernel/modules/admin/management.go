package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/adminui"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/core/access"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/group"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/modules/resourceextension"
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
	UserReadPermission,
	UserCreatePermission,
	UserUpdatePermission,
	UserBlockPermission,
	GroupReadPermission,
	GroupCreatePermission,
	GroupUpdatePermission,
	GroupDeletePermission,
	FileReadPermission,
	FileCreatePermission,
	FileUpdatePermission,
	FileDeletePermission,
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
	ProfileBlueprint(kernel.ProfileCode) (*kernel.ProfileBlueprint, bool)
}

type SiteCatalog interface {
	RuntimeByID(site.ID) (*site.Runtime, bool)
	Create(context.Context, security.Actor, site.CreateInput) (*site.Runtime, error)
	Update(context.Context, security.Actor, site.UpdateInput) (*site.Runtime, error)
	Delete(context.Context, security.Actor, site.ID) error
}

type Management struct {
	profiles      []kernel.Profile
	profileSource ProfileResolver
	repository    site.ManagementRepository
	sites         SiteCatalog
	resources     *resource.Service
	resourceRepo  resource.ManagementRepository
	authorizer    security.Authorizer
	policy        SiteAccessPolicy
	users         user.Service
	userRepo      user.ManagementRepository
	groups        group.Service
	groupRepo     group.ManagementRepository
	access        access.Service
	files         file.ManagementService
	media         media.Service
	maxUploadSize int64
	uploadTimeout time.Duration
	avatarStorage filesystem.Code
	avatarMaxSize int64
	navigation    *navigationComposer
}

type ManagementDependencies struct {
	Profiles           []kernel.Profile
	ProfileSource      ProfileResolver
	SiteRepository     site.ManagementRepository
	Sites              SiteCatalog
	Resources          *resource.Service
	ResourceRepository resource.ManagementRepository
	Authorizer         security.Authorizer
	SiteAccessPolicy   SiteAccessPolicy
	Users              user.Service
	UserRepository     user.ManagementRepository
	Groups             group.Service
	GroupRepository    group.ManagementRepository
	Access             access.Service
	Files              file.ManagementService
	Media              media.Service
	MaxUploadSize      int64
	UploadTimeout      time.Duration
	AvatarStorage      filesystem.Code
	AvatarMaxSize      int64
	Permissions        adminui.PermissionValidator
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
	if dependencies.Permissions == nil {
		return nil, errors.New("admin permission catalog is nil")
	}
	if dependencies.SiteAccessPolicy == nil {
		return nil, errors.New("admin site access policy is nil")
	}
	if dependencies.Users == nil || dependencies.UserRepository == nil {
		return nil, errors.New("admin user dependencies are nil")
	}
	if dependencies.Groups == nil || dependencies.GroupRepository == nil {
		return nil, errors.New("admin group dependencies are nil")
	}
	if dependencies.Access == nil {
		return nil, errors.New("admin access service is nil")
	}
	if dependencies.Files == nil {
		return nil, errors.New("admin file service is nil")
	}
	if dependencies.Media == nil {
		return nil, errors.New("admin media service is nil")
	}
	if dependencies.MaxUploadSize <= 0 {
		dependencies.MaxUploadSize = 100 << 20
	}
	if dependencies.UploadTimeout <= 0 {
		dependencies.UploadTimeout = 10 * time.Minute
	}
	if dependencies.AvatarStorage == "" {
		dependencies.AvatarStorage = "private"
	}
	if dependencies.AvatarMaxSize <= 0 {
		dependencies.AvatarMaxSize = 5 << 20
	}
	navigation, err := newNavigationComposer(
		dependencies.Profiles,
		dependencies.Authorizer,
		dependencies.Permissions,
	)
	if err != nil {
		return nil, err
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
		users:         dependencies.Users,
		userRepo:      dependencies.UserRepository,
		groups:        dependencies.Groups,
		groupRepo:     dependencies.GroupRepository,
		access:        dependencies.Access,
		files:         dependencies.Files,
		media:         dependencies.Media,
		maxUploadSize: dependencies.MaxUploadSize,
		uploadTimeout: dependencies.UploadTimeout,
		avatarStorage: dependencies.AvatarStorage,
		avatarMaxSize: dependencies.AvatarMaxSize,
		navigation:    navigation,
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
	Code   kernel.ProfileCode `json:"code"`
	Name   string             `json:"name"`
	Fields []FieldDefinition  `json:"fields"`
}

type SiteProfiles struct {
	Items []SiteProfile `json:"items"`
}

func (m *Management) Navigation(
	ctx context.Context,
	actor security.Actor,
	selectedSiteID *site.ID,
) (Navigation, error) {
	if m == nil || m.navigation == nil {
		return Navigation{}, errors.New("admin navigation is unavailable")
	}
	var runtime *site.Runtime
	if selectedSiteID != nil {
		if err := m.requireSite(
			ctx,
			actor,
			*selectedSiteID,
			SiteReadPermission,
		); err != nil {
			return Navigation{}, err
		}
		var exists bool
		runtime, exists = m.sites.RuntimeByID(*selectedSiteID)
		if !exists {
			return Navigation{}, site.ErrNotFound
		}
	}

	items, err := m.navigation.compose(
		ctx,
		actor,
		runtime,
	)
	if err != nil {
		return Navigation{}, err
	}
	return Navigation{Items: navigationDTO(items)}, nil
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
	Settings    map[string]any
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
	_, err = m.resources.Create(ctx, security.System(), resource.CreateInput{
		SiteID:   runtime.Site().ID,
		Type:     resourcetype.Page,
		Title:    "Первая страница",
		Slug:     "",
		Settings: map[string]any{},
	})
	if err != nil {
		rollbackErr := m.sites.Delete(ctx, security.System(), runtime.Site().ID)
		if rollbackErr != nil {
			return SiteDetails{}, fmt.Errorf(
				"create initial resource: %v; rollback site: %w",
				err,
				rollbackErr,
			)
		}
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
	if _, exists := m.sites.RuntimeByID(id); !exists {
		return SiteDetails{}, site.ErrNotFound
	}
	runtime, err := m.sites.Update(ctx, actor, site.UpdateInput{
		ID:          id,
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
		blueprint, exists := m.profileSource.ProfileBlueprint(profile.Code)
		if !exists {
			continue
		}
		fields, err := fieldDefinitions(blueprint.ParamSchema().Definitions())
		if err != nil {
			return SiteProfiles{}, err
		}
		items = append(items, SiteProfile{
			Code:   profile.Code,
			Name:   profile.Name,
			Fields: fields,
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
	Sort           int            `json:"sort"`
	Deleted        bool           `json:"deleted"`
	Published      bool           `json:"published"`
	DeletedAt      *time.Time     `json:"deleted_at"`
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
	Code                    template.Code     `json:"code"`
	Label                   string            `json:"label"`
	Icon                    string            `json:"icon"`
	Fields                  []FieldDefinition `json:"fields"`
	SupportsResourceWidgets bool              `json:"supports_resource_widgets"`
	WidgetAreas             []widget.AreaCode `json:"widget_areas"`
}

type WidgetEditorTab struct {
	Code   string   `json:"code"`
	Label  string   `json:"label"`
	Fields []string `json:"fields"`
}

type WidgetView struct {
	Code  widget.ViewCode `json:"code"`
	Label string          `json:"label"`
}

type WidgetDefinition struct {
	Code              widget.Code       `json:"code"`
	ModuleCode        string            `json:"module_code"`
	ModuleLabel       string            `json:"module_label"`
	ModuleDescription string            `json:"module_description"`
	Label             string            `json:"label"`
	Description       string            `json:"description"`
	Fields            []FieldDefinition `json:"fields"`
	EditorTabs        []WidgetEditorTab `json:"editor_tabs"`
	SummaryFields     []string          `json:"summary_fields"`
	Views             []WidgetView      `json:"views"`
}

type ResourceType struct {
	Code  resourcetype.Code `json:"code"`
	Label string            `json:"label"`
}

type ResourceMetadata struct {
	Types      []ResourceType               `json:"types"`
	Templates  []ResourceTemplate           `json:"templates"`
	Widgets    []WidgetDefinition           `json:"widgets"`
	Extensions []resourceextension.Metadata `json:"extensions"`
}

func (m *Management) ResourceMetadata(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
) (ResourceMetadata, error) {
	if err := m.requireSite(ctx, actor, siteID, ResourceReadPermission); err != nil {
		return ResourceMetadata{}, err
	}
	runtime, exists := m.sites.RuntimeByID(siteID)
	if !exists {
		return ResourceMetadata{}, site.ErrNotFound
	}
	definitions := runtime.Profile().Templates()
	templates := make([]ResourceTemplate, len(definitions))
	for index, definition := range definitions {
		fields, err := fieldDefinitions(definition.Fields)
		if err != nil {
			return ResourceMetadata{}, err
		}
		templateRuntime, _ := runtime.Profile().Template(definition.Code)
		templates[index] = ResourceTemplate{
			Code:                    definition.Code,
			Label:                   definition.Label,
			Icon:                    iconOrDefault(definition.Icon),
			Fields:                  fields,
			SupportsResourceWidgets: templateRuntime.SupportsResourceWidgets(),
			WidgetAreas:             templateRuntime.ResourceAreas(),
		}
	}
	widgetDefinitions := runtime.Profile().Widgets()
	widgets := make([]WidgetDefinition, len(widgetDefinitions))
	for index, definition := range widgetDefinitions {
		fields, err := fieldDefinitions(definition.Fields)
		if err != nil {
			return ResourceMetadata{}, err
		}
		tabs := make([]WidgetEditorTab, len(definition.EditorTabs))
		for tabIndex, tab := range definition.EditorTabs {
			tabs[tabIndex] = WidgetEditorTab{Code: tab.Code, Label: tab.Label, Fields: append([]string(nil), tab.Fields...)}
		}
		views := make([]WidgetView, len(definition.Views))
		for viewIndex, view := range definition.Views {
			views[viewIndex] = WidgetView{Code: view.Code(), Label: view.Label()}
		}
		widgets[index] = WidgetDefinition{
			Code: definition.Code, ModuleCode: definition.Module.Code,
			ModuleLabel: definition.Module.Label, ModuleDescription: definition.Module.Description,
			Label: definition.Label, Description: definition.Description, Fields: fields,
			EditorTabs: tabs, SummaryFields: append([]string{}, definition.SummaryFields...), Views: views,
		}
	}
	types := []ResourceType{
		{Code: resourcetype.Page, Label: "Страница"},
		{Code: resourcetype.Link, Label: "Ссылка"},
	}
	extensions := make([]resourceextension.Metadata, 0)
	for _, moduleRuntime := range runtime.Profile().Modules() {
		provider, ok := moduleRuntime.(resourceextension.EditorProvider)
		if !ok {
			continue
		}
		editor := provider.ResourceEditorExtension()
		if editor == nil {
			return ResourceMetadata{}, errors.New(
				"resource editor extension is nil",
			)
		}
		extensions = append(extensions, editor.Metadata())
	}
	return ResourceMetadata{
		Types: types, Templates: templates, Widgets: widgets, Extensions: extensions,
	}, nil
}

func (m *Management) ResourceExtension(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	code resourceextension.Code,
) (any, error) {
	editor, request, err := m.resourceEditorExtension(
		ctx, actor, siteID, resourceID, code, ResourceReadPermission,
	)
	if err != nil {
		return nil, err
	}
	result, err := editor.Read(ctx, request)
	return result, resourceExtensionError(err)
}

func (m *Management) SaveResourceExtension(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	code resourceextension.Code,
	payload json.RawMessage,
) (any, error) {
	editor, request, err := m.resourceEditorExtension(
		ctx, actor, siteID, resourceID, code, ResourceUpdatePermission,
	)
	if err != nil {
		return nil, err
	}
	result, err := editor.Save(ctx, request, payload)
	return result, resourceExtensionError(err)
}

func (m *Management) PreviewResourceExtension(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	code resourceextension.Code,
	payload json.RawMessage,
) (any, error) {
	editor, request, err := m.resourceEditorExtension(
		ctx, actor, siteID, resourceID, code, ResourceReadPermission,
	)
	if err != nil {
		return nil, err
	}
	result, err := editor.Preview(ctx, request, payload)
	return result, resourceExtensionError(err)
}

func (m *Management) resourceEditorExtension(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	code resourceextension.Code,
	permissionCode permission.Code,
) (resourceextension.Editor, resourceextension.Request, error) {
	if code == "" {
		return nil, resourceextension.Request{}, resource.ErrNotFound
	}
	if err := m.requireSite(ctx, actor, siteID, permissionCode); err != nil {
		return nil, resourceextension.Request{}, err
	}
	item, err := m.resourceRepo.ByID(ctx, resourceID)
	if err != nil {
		return nil, resourceextension.Request{}, err
	}
	if item.SiteID != siteID {
		return nil, resourceextension.Request{}, resource.ErrNotFound
	}
	siteRuntime, exists := m.sites.RuntimeByID(siteID)
	if !exists {
		return nil, resourceextension.Request{}, site.ErrNotFound
	}
	for _, moduleRuntime := range siteRuntime.Profile().Modules() {
		provider, ok := moduleRuntime.(resourceextension.EditorProvider)
		if !ok {
			continue
		}
		editor := provider.ResourceEditorExtension()
		if editor == nil || editor.Metadata().Code != code {
			continue
		}
		if !editor.AppliesTo(item.Type) {
			return nil, resourceextension.Request{}, resource.ErrNotFound
		}
		return editor, resourceextension.Request{
			Actor: actor, Site: siteRuntime.Site(), Resource: item,
		}, nil
	}
	return nil, resourceextension.Request{}, resource.ErrNotFound
}

func resourceExtensionError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, resourceextension.ErrNotApplicable) {
		return resource.ErrNotFound
	}
	var validation resourceextension.ValidationError
	if !errors.As(err, &validation) {
		return err
	}
	fields := make([]FieldValidationError, len(validation.Fields))
	for index, field := range validation.Fields {
		fields[index] = FieldValidationError{
			Key: field.Key, Rule: "extension", Param: field.Message,
		}
	}
	return ValidationError{Message: validation.Error(), Fields: fields}
}

type ResourceCreateInput struct {
	ParentID    *resource.ID
	Type        resourcetype.Code
	Template    *template.Code
	Title       string
	MenuTitle   string
	Slug        string
	ExternalURL *string
	Settings    map[string]any
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
		Settings:    input.Settings,
	})
	if err != nil {
		if errors.Is(err, resource.ErrNotFound) {
			return ResourceTreeItem{}, err
		}
		return ResourceTreeItem{}, validationError(err)
	}
	return treeItem(runtime, resource.Child{
		ID:            created.ID,
		SiteID:        created.SiteID,
		ParentID:      created.ParentID,
		Type:          created.Type,
		Template:      created.Template,
		Title:         created.Title,
		MenuTitle:     created.MenuTitle,
		Sort:          created.Sort,
		IsPublic:      created.IsPublic,
		PublishedAt:   created.PublishedAt,
		UnpublishedAt: created.UnpublishedAt,
		DeletedAt:     created.DeletedAt,
	}, true), nil
}

type ResourceDTO struct {
	ID            resource.ID       `json:"id"`
	SiteID        site.ID           `json:"site_id"`
	ParentID      *resource.ID      `json:"parent_id"`
	Type          resourcetype.Code `json:"type"`
	TemplateCode  *template.Code    `json:"template_code"`
	Title         string            `json:"title"`
	MenuTitle     string            `json:"menu_title"`
	Slug          string            `json:"slug"`
	Path          *string           `json:"path"`
	Annotation    string            `json:"annotation"`
	ContentType   *string           `json:"content_type"`
	Content       string            `json:"content"`
	ExternalURL   *string           `json:"external_url"`
	IsPublic      bool              `json:"is_public"`
	IsSearchable  bool              `json:"is_searchable"`
	InMenu        bool              `json:"in_menu"`
	InSitemap     bool              `json:"in_sitemap"`
	Sort          int               `json:"sort"`
	PublishedAt   *time.Time        `json:"published_at"`
	UnpublishedAt *time.Time        `json:"unpublished_at"`
	Deleted       bool              `json:"deleted"`
	DeletedAt     *time.Time        `json:"deleted_at"`
	Settings      map[string]any    `json:"settings"`
	Widgets       []ResourceWidget  `json:"widgets"`
}

type ResourceWidget struct {
	ID           widget.BindingID `json:"id"`
	Code         widget.Code      `json:"code"`
	Area         widget.AreaCode  `json:"area"`
	Position     int              `json:"position"`
	View         widget.ViewCode  `json:"view"`
	Columns      int              `json:"columns"`
	MarginTop    int              `json:"margin_top"`
	MarginBottom int              `json:"margin_bottom"`
	Enabled      bool             `json:"enabled"`
	Params       map[string]any   `json:"params"`
}

type ResourceDetails struct {
	Resource    ResourceDTO `json:"resource"`
	Permissions struct {
		Update  bool `json:"update"`
		Delete  bool `json:"delete"`
		Restore bool `json:"restore"`
	} `json:"permissions"`
}

type ResourceUpdateInput struct {
	ParentID      *resource.ID
	Type          resourcetype.Code
	Template      *template.Code
	Title         string
	MenuTitle     string
	Slug          string
	Annotation    string
	ContentType   *string
	Content       string
	ExternalURL   *string
	IsPublic      bool
	IsSearchable  bool
	InMenu        bool
	InSitemap     bool
	Sort          int
	PublishedAt   *time.Time
	UnpublishedAt *time.Time
	Settings      map[string]any
}

func (m *Management) Resource(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
) (ResourceDetails, error) {
	if err := m.requireSite(ctx, actor, siteID, ResourceReadPermission); err != nil {
		return ResourceDetails{}, err
	}
	item, err := m.resources.Get(ctx, actor, resourceID)
	if err != nil {
		return ResourceDetails{}, err
	}
	if item.SiteID != siteID {
		return ResourceDetails{}, resource.ErrNotFound
	}
	canUpdate, err := m.allowed(ctx, actor, ResourceUpdatePermission)
	if err != nil {
		return ResourceDetails{}, err
	}
	result := ResourceDetails{Resource: resourceDTO(item)}
	result.Permissions.Update = canUpdate
	canDelete, err := m.allowed(ctx, actor, ResourceDeletePermission)
	if err != nil {
		return ResourceDetails{}, err
	}
	result.Permissions.Delete = canDelete
	result.Permissions.Restore = canDelete
	if item.ParentID != nil {
		parent, parentErr := m.resources.Get(ctx, actor, *item.ParentID)
		if parentErr != nil {
			return ResourceDetails{}, parentErr
		}
		result.Permissions.Restore = canDelete && parent.DeletedAt == nil
	}
	return result, nil
}

func (m *Management) UpdateResource(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	input ResourceUpdateInput,
) (ResourceDetails, error) {
	if err := m.requireSite(ctx, actor, siteID, ResourceUpdatePermission); err != nil {
		return ResourceDetails{}, err
	}
	current, err := m.resources.Get(ctx, actor, resourceID)
	if err != nil {
		return ResourceDetails{}, err
	}
	if current.SiteID != siteID {
		return ResourceDetails{}, resource.ErrNotFound
	}
	if current.Type != resourcetype.Page && current.Type != resourcetype.Link {
		return ResourceDetails{}, fmt.Errorf("%w: unsupported current resource type", ErrValidation)
	}
	if input.Type != resourcetype.Page && input.Type != resourcetype.Link {
		return ResourceDetails{}, fmt.Errorf("%w: unsupported resource type", ErrValidation)
	}

	var contentType *string
	if input.Type == resourcetype.Page {
		value := "html"
		contentType = &value
	}
	updated, err := m.resources.Update(ctx, actor, resource.UpdateInput{
		ID:               resourceID,
		ParentID:         input.ParentID,
		Type:             input.Type,
		Template:         input.Template,
		ContentType:      contentType,
		Title:            input.Title,
		MenuTitle:        input.MenuTitle,
		Slug:             input.Slug,
		Annotation:       input.Annotation,
		Content:          input.Content,
		ImageMediaID:     current.ImageMediaID,
		TargetResourceID: nil,
		ExternalURL:      input.ExternalURL,
		IsPublic:         input.IsPublic,
		IsSearchable:     input.IsSearchable,
		InMenu:           input.InMenu,
		InSitemap:        input.InSitemap,
		Sort:             input.Sort,
		PublishedAt:      input.PublishedAt,
		UnpublishedAt:    input.UnpublishedAt,
		Settings:         input.Settings,
	})
	if err != nil {
		return ResourceDetails{}, validationError(err)
	}
	result := ResourceDetails{Resource: resourceDTO(updated)}
	result.Permissions.Update = true
	canDelete, permissionErr := m.allowed(ctx, actor, ResourceDeletePermission)
	if permissionErr != nil {
		return ResourceDetails{}, permissionErr
	}
	result.Permissions.Delete = canDelete
	result.Permissions.Restore = canDelete
	return result, nil
}

func (m *Management) CreateResourceWidget(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	input resource.CreateWidgetInput,
) (ResourceWidget, error) {
	if err := m.requireSite(ctx, actor, siteID, ResourceUpdatePermission); err != nil {
		return ResourceWidget{}, err
	}
	if err := m.requireResourceSite(ctx, actor, siteID, resourceID); err != nil {
		return ResourceWidget{}, err
	}
	created, err := m.resources.CreateWidget(ctx, actor, resourceID, input)
	if err != nil {
		return ResourceWidget{}, validationError(err)
	}
	return resourceWidgets([]widget.Binding{created})[0], nil
}

func (m *Management) UpdateResourceWidget(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	bindingID widget.BindingID,
	input resource.UpdateWidgetInput,
) (ResourceWidget, error) {
	if err := m.requireSite(ctx, actor, siteID, ResourceUpdatePermission); err != nil {
		return ResourceWidget{}, err
	}
	if err := m.requireResourceSite(ctx, actor, siteID, resourceID); err != nil {
		return ResourceWidget{}, err
	}
	updated, err := m.resources.UpdateWidget(ctx, actor, resourceID, bindingID, input)
	if err != nil {
		if errors.Is(err, resource.ErrNotFound) {
			return ResourceWidget{}, err
		}
		return ResourceWidget{}, validationError(err)
	}
	return resourceWidgets([]widget.Binding{updated})[0], nil
}

func (m *Management) DeleteResourceWidget(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	bindingID widget.BindingID,
) error {
	if err := m.requireSite(ctx, actor, siteID, ResourceUpdatePermission); err != nil {
		return err
	}
	if err := m.requireResourceSite(ctx, actor, siteID, resourceID); err != nil {
		return err
	}
	return m.resources.DeleteWidget(ctx, actor, resourceID, bindingID)
}

func (m *Management) ReorderResourceWidgets(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	order []widget.Order,
) ([]ResourceWidget, error) {
	if err := m.requireSite(ctx, actor, siteID, ResourceUpdatePermission); err != nil {
		return nil, err
	}
	if err := m.requireResourceSite(ctx, actor, siteID, resourceID); err != nil {
		return nil, err
	}
	updated, err := m.resources.ReorderWidgets(ctx, actor, resourceID, order)
	if err != nil {
		return nil, validationError(err)
	}
	return resourceWidgets(updated), nil
}

func (m *Management) requireResourceSite(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
) error {
	current, err := m.resources.Get(ctx, actor, resourceID)
	if err != nil {
		return err
	}
	if current.SiteID != siteID {
		return resource.ErrNotFound
	}
	return nil
}

func (m *Management) MoveResource(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	parentID *resource.ID,
	position int,
) (ResourceDTO, error) {
	if err := m.requireSite(ctx, actor, siteID, ResourceUpdatePermission); err != nil {
		return ResourceDTO{}, err
	}
	current, err := m.resources.Get(ctx, actor, resourceID)
	if err != nil {
		return ResourceDTO{}, err
	}
	if current.SiteID != siteID {
		return ResourceDTO{}, resource.ErrNotFound
	}
	if parentID != nil {
		parent, parentErr := m.resources.Get(ctx, actor, *parentID)
		if parentErr != nil {
			return ResourceDTO{}, parentErr
		}
		if parent.SiteID != siteID {
			return ResourceDTO{}, resource.ErrInvalidTree
		}
	}
	updated, err := m.resources.Move(ctx, actor, resourceID, parentID, position)
	if err != nil {
		return ResourceDTO{}, err
	}
	return resourceDTO(updated), nil
}

func (m *Management) DeleteResource(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	permanent bool,
) error {
	if err := m.requireSite(ctx, actor, siteID, ResourceDeletePermission); err != nil {
		return err
	}
	current, err := m.resources.Get(ctx, actor, resourceID)
	if err != nil {
		return err
	}
	if current.SiteID != siteID {
		return resource.ErrNotFound
	}
	if permanent {
		return m.resources.DeletePermanent(ctx, actor, resourceID)
	}
	return m.resources.Delete(ctx, actor, resourceID)
}

func (m *Management) RestoreResource(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
	resourceID resource.ID,
	withDescendants bool,
) error {
	if err := m.requireSite(ctx, actor, siteID, ResourceDeletePermission); err != nil {
		return err
	}
	current, err := m.resources.Get(ctx, actor, resourceID)
	if err != nil {
		return err
	}
	if current.SiteID != siteID {
		return resource.ErrNotFound
	}
	return m.resources.Restore(ctx, actor, resourceID, withDescendants)
}

type ResourceOption struct {
	ID           resource.ID  `json:"id"`
	ParentID     *resource.ID `json:"parent_id"`
	DisplayTitle string       `json:"display_title"`
	Path         *string      `json:"path"`
}

type ResourceOptions struct {
	Items []ResourceOption `json:"items"`
}

func (m *Management) ResourceOptions(
	ctx context.Context,
	actor security.Actor,
	siteID site.ID,
) (ResourceOptions, error) {
	if err := m.requireSite(ctx, actor, siteID, ResourceReadPermission); err != nil {
		return ResourceOptions{}, err
	}
	tree, err := m.resources.Tree(ctx, actor, siteID)
	if err != nil {
		return ResourceOptions{}, err
	}
	items := make([]ResourceOption, 0)
	appendResourceOptions(&items, tree)
	return ResourceOptions{Items: items}, nil
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
		Sort:           item.Sort,
		Deleted:        item.DeletedAt != nil,
		Published:      isPublished(item),
		DeletedAt:      item.DeletedAt,
		HasChildren:    item.HasChildren,
		CanCreateChild: canCreate && item.DeletedAt == nil,
	}
}

func isPublished(item resource.Child) bool {
	if item.DeletedAt != nil || !item.IsPublic {
		return false
	}
	now := time.Now().UTC()
	return (item.PublishedAt == nil || !now.Before(item.PublishedAt.UTC())) &&
		(item.UnpublishedAt == nil || now.Before(item.UnpublishedAt.UTC()))
}

func resourceDTO(item resource.Resource) ResourceDTO {
	settings := make(map[string]any, len(item.Settings))
	for key, value := range item.Settings {
		settings[key] = value
	}
	return ResourceDTO{
		ID:            item.ID,
		SiteID:        item.SiteID,
		ParentID:      item.ParentID,
		Type:          item.Type,
		TemplateCode:  item.Template,
		Title:         item.Title,
		MenuTitle:     item.MenuTitle,
		Slug:          item.Slug,
		Path:          item.Path,
		Annotation:    item.Annotation,
		ContentType:   item.ContentType,
		Content:       item.Content,
		ExternalURL:   item.ExternalURL,
		IsPublic:      item.IsPublic,
		IsSearchable:  item.IsSearchable,
		InMenu:        item.InMenu,
		InSitemap:     item.InSitemap,
		Sort:          item.Sort,
		PublishedAt:   item.PublishedAt,
		UnpublishedAt: item.UnpublishedAt,
		Deleted:       item.DeletedAt != nil,
		DeletedAt:     item.DeletedAt,
		Settings:      settings,
		Widgets:       resourceWidgets(item.Widgets),
	}
}

func resourceWidgets(source []widget.Binding) []ResourceWidget {
	result := make([]ResourceWidget, len(source))
	for index, binding := range source {
		params := make(map[string]any, len(binding.Params))
		for key, value := range binding.Params {
			params[key] = value
		}
		result[index] = ResourceWidget{
			ID: binding.ID, Code: binding.Code, Area: binding.Area, Position: binding.Position,
			View: widget.PublicView(binding.Presentation.View), Columns: binding.Presentation.Columns,
			MarginTop: binding.Presentation.MarginTop, MarginBottom: binding.Presentation.MarginBottom,
			Enabled: binding.Presentation.Enabled, Params: params,
		}
	}
	return result
}

func appendResourceOptions(target *[]ResourceOption, nodes []resource.Node) {
	for _, node := range nodes {
		item := node.Resource
		if item.DeletedAt != nil {
			continue
		}
		displayTitle := strings.TrimSpace(item.MenuTitle)
		if displayTitle == "" {
			displayTitle = item.Title
		}
		*target = append(*target, ResourceOption{
			ID:           item.ID,
			ParentID:     item.ParentID,
			DisplayTitle: displayTitle,
			Path:         item.Path,
		})
		appendResourceOptions(target, node.Children)
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
	if errors.Is(err, security.ErrUnauthenticated) || errors.Is(err, security.ErrForbidden) {
		return err
	}
	if errors.Is(err, site.ErrConflict) || errors.Is(err, site.ErrNotFound) ||
		errors.Is(err, resource.ErrConflict) || errors.Is(err, resource.ErrNotFound) {
		return err
	}
	var fieldErrors field.ValidationErrors
	if errors.As(err, &fieldErrors) {
		fields := make([]FieldValidationError, len(fieldErrors))
		for index, item := range fieldErrors {
			fields[index] = FieldValidationError{
				Key: item.Key, Rule: item.Rule, Param: item.Param,
			}
		}
		return ValidationError{
			Message: "request data is invalid",
			Fields:  fields,
		}
	}
	if errors.Is(err, site.ErrInvalid) || errors.Is(err, resource.ErrInvalid) ||
		errors.Is(err, resource.ErrInvalidReference) {
		return fmt.Errorf("%w: request data is invalid", ErrValidation)
	}
	return err
}
