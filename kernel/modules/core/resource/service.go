package resource

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

type SiteResolver interface {
	RuntimeByID(site.ID) (*site.Runtime, bool)
}

type Service struct {
	repository Repository
	sites      SiteResolver
}

func NewService(
	repository Repository,
	sites SiteResolver,
) (*Service, error) {
	if repository == nil {
		return nil, errors.New("resource repository is nil")
	}
	if sites == nil {
		return nil, errors.New("resource site resolver is nil")
	}

	return &Service{
		repository: repository,
		sites:      sites,
	}, nil
}

func (s *Service) Create(
	ctx context.Context,
	input CreateInput,
) (Resource, error) {
	if err := validateContext(ctx, "resource create"); err != nil {
		return Resource{}, err
	}
	if input.SiteID <= 0 {
		return Resource{}, errors.New("resource site id is invalid")
	}

	siteRuntime, exists := s.sites.RuntimeByID(input.SiteID)
	if !exists {
		return Resource{}, fmt.Errorf(
			"resource site %d not found",
			input.SiteID,
		)
	}

	resourceType := input.Type
	if resourceType == "" {
		resourceType = resourcetype.Page
	}

	item := Resource{
		SiteID:           input.SiteID,
		ParentID:         cloneID(input.ParentID),
		Type:             resourceType,
		Template:         cloneTemplateCode(input.Template),
		ContentType:      cloneString(input.ContentType),
		Title:            input.Title,
		MenuTitle:        input.MenuTitle,
		Slug:             input.Slug,
		Content:          input.Content,
		TargetResourceID: cloneID(input.TargetResourceID),
		ExternalURL:      cloneString(input.ExternalURL),
		IsPublic:         boolDefault(input.IsPublic, true),
		IsSearchable:     boolDefault(input.IsSearchable, true),
		InMenu:           boolDefault(input.InMenu, true),
		InSitemap:        boolDefault(input.InSitemap, true),
		Sort:             input.Sort,
		PublishedAt:      cloneTime(input.PublishedAt),
		UnpublishedAt:    cloneTime(input.UnpublishedAt),
		Settings:         cloneMap(input.Settings),
	}

	normalized, err := s.normalize(
		ctx,
		item,
		siteRuntime,
		nil,
	)
	if err != nil {
		return Resource{}, err
	}

	created, err := s.repository.Create(ctx, normalized)
	if err != nil {
		return Resource{}, fmt.Errorf("create resource: %w", err)
	}

	return s.validateStored(ctx, created)
}

func (s *Service) Get(
	ctx context.Context,
	id ID,
) (Resource, error) {
	if err := validateContext(ctx, "resource get"); err != nil {
		return Resource{}, err
	}
	if id <= 0 {
		return Resource{}, errors.New("resource id is invalid")
	}

	item, err := s.repository.ByID(ctx, id)
	if err != nil {
		return Resource{}, fmt.Errorf("get resource %d: %w", id, err)
	}

	return s.validateStored(ctx, item)
}

func (s *Service) GetByPath(
	ctx context.Context,
	siteID site.ID,
	path string,
) (Resource, error) {
	if err := validateContext(ctx, "resource get by path"); err != nil {
		return Resource{}, err
	}
	if siteID <= 0 {
		return Resource{}, errors.New("resource site id is invalid")
	}
	if !validLookupPath(path) {
		return Resource{}, fmt.Errorf(
			"resource path %q is invalid",
			path,
		)
	}

	item, err := s.repository.ByPath(ctx, siteID, path)
	if err != nil {
		return Resource{}, fmt.Errorf(
			"get resource by path %q: %w",
			path,
			err,
		)
	}

	return s.validateStored(ctx, item)
}

func (s *Service) Update(
	ctx context.Context,
	input UpdateInput,
) (Resource, error) {
	if err := validateContext(ctx, "resource update"); err != nil {
		return Resource{}, err
	}
	if input.ID <= 0 {
		return Resource{}, errors.New("resource id is invalid")
	}
	if input.Type == "" {
		return Resource{}, errors.New("resource type is empty")
	}

	current, err := s.repository.ByID(ctx, input.ID)
	if err != nil {
		return Resource{}, fmt.Errorf(
			"get resource %d for update: %w",
			input.ID,
			err,
		)
	}
	current, err = s.validateStored(ctx, current)
	if err != nil {
		return Resource{}, fmt.Errorf(
			"validate resource %d for update: %w",
			input.ID,
			err,
		)
	}
	siteRuntime, exists := s.sites.RuntimeByID(current.SiteID)
	if !exists {
		return Resource{}, fmt.Errorf(
			"resource site %d not found",
			current.SiteID,
		)
	}

	item := Resource{
		ID:               current.ID,
		SiteID:           current.SiteID,
		ParentID:         cloneID(input.ParentID),
		Type:             input.Type,
		Template:         cloneTemplateCode(input.Template),
		ContentType:      cloneString(input.ContentType),
		Title:            input.Title,
		MenuTitle:        input.MenuTitle,
		Slug:             input.Slug,
		Content:          input.Content,
		TargetResourceID: cloneID(input.TargetResourceID),
		ExternalURL:      cloneString(input.ExternalURL),
		IsPublic:         input.IsPublic,
		IsSearchable:     input.IsSearchable,
		InMenu:           input.InMenu,
		InSitemap:        input.InSitemap,
		Sort:             input.Sort,
		PublishedAt:      cloneTime(input.PublishedAt),
		UnpublishedAt:    cloneTime(input.UnpublishedAt),
		Settings:         cloneMap(input.Settings),
		CreatedAt:        current.CreatedAt,
		UpdatedAt:        current.UpdatedAt,
	}

	if err := s.ensureNoParentCycle(ctx, item); err != nil {
		return Resource{}, err
	}

	normalized, err := s.normalize(
		ctx,
		item,
		siteRuntime,
		nil,
	)
	if err != nil {
		return Resource{}, err
	}
	if current.Path != nil && normalized.Path == nil {
		if err := s.ensureNoRouteDescendants(
			ctx,
			current,
			siteRuntime,
		); err != nil {
			return Resource{}, err
		}
	}

	updated, err := s.repository.Update(ctx, normalized)
	if err != nil {
		return Resource{}, fmt.Errorf(
			"update resource %d: %w",
			input.ID,
			err,
		)
	}

	return s.validateStored(ctx, updated)
}

func (s *Service) Delete(
	ctx context.Context,
	id ID,
) error {
	if err := validateContext(ctx, "resource delete"); err != nil {
		return err
	}
	if id <= 0 {
		return errors.New("resource id is invalid")
	}

	if err := s.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete resource %d: %w", id, err)
	}
	return nil
}

func (s *Service) Tree(
	ctx context.Context,
	siteID site.ID,
) ([]Node, error) {
	if err := validateContext(ctx, "resource tree"); err != nil {
		return nil, err
	}
	if siteID <= 0 {
		return nil, errors.New("resource site id is invalid")
	}

	siteRuntime, exists := s.sites.RuntimeByID(siteID)
	if !exists {
		return nil, fmt.Errorf("resource site %d not found", siteID)
	}

	items, err := s.repository.ListBySite(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("list resources for site %d: %w", siteID, err)
	}

	rawByID := make(map[ID]Resource, len(items))
	for index, item := range items {
		if item.ID <= 0 {
			return nil, fmt.Errorf(
				"resource at index %d has invalid id",
				index,
			)
		}
		if item.SiteID != siteID {
			return nil, fmt.Errorf(
				"resource %d belongs to site %d instead of %d",
				item.ID,
				item.SiteID,
				siteID,
			)
		}
		if _, exists := rawByID[item.ID]; exists {
			return nil, fmt.Errorf(
				"duplicate resource id %d",
				item.ID,
			)
		}
		rawByID[item.ID] = item
	}

	normalized := make([]Resource, 0, len(items))
	for _, item := range items {
		storedPath := cloneString(item.Path)
		result, err := s.normalize(
			ctx,
			item,
			siteRuntime,
			rawByID,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"validate stored resource %d: %w",
				item.ID,
				err,
			)
		}
		if !equalStrings(storedPath, result.Path) {
			return nil, fmt.Errorf(
				"validate stored resource %d: stored path is inconsistent",
				item.ID,
			)
		}
		normalized = append(normalized, result)
	}

	return buildTree(normalized)
}

func (s *Service) validateStored(
	ctx context.Context,
	item Resource,
) (Resource, error) {
	if item.ID <= 0 {
		return Resource{}, errors.New("stored resource id is invalid")
	}
	if item.SiteID <= 0 {
		return Resource{}, errors.New("stored resource site id is invalid")
	}

	siteRuntime, exists := s.sites.RuntimeByID(item.SiteID)
	if !exists {
		return Resource{}, fmt.Errorf(
			"stored resource %d references unknown site %d",
			item.ID,
			item.SiteID,
		)
	}

	storedPath := cloneString(item.Path)
	normalized, err := s.normalize(ctx, item, siteRuntime, nil)
	if err != nil {
		return Resource{}, err
	}
	if !equalStrings(storedPath, normalized.Path) {
		return Resource{}, fmt.Errorf(
			"stored resource %d path is inconsistent",
			item.ID,
		)
	}
	return normalized, nil
}

func (s *Service) normalize(
	ctx context.Context,
	item Resource,
	siteRuntime *site.Runtime,
	known map[ID]Resource,
) (Resource, error) {
	item = Clone(item)
	item.Title = strings.TrimSpace(item.Title)
	item.MenuTitle = strings.TrimSpace(item.MenuTitle)

	if item.Title == "" {
		return Resource{}, errors.New("resource title is empty")
	}
	if !validSlug(item.Slug, item.ParentID) {
		return Resource{}, fmt.Errorf(
			"resource slug %q is invalid",
			item.Slug,
		)
	}
	if item.PublishedAt != nil &&
		item.UnpublishedAt != nil &&
		!item.UnpublishedAt.After(*item.PublishedAt) {
		return Resource{}, errors.New(
			"resource unpublished_at must be after published_at",
		)
	}

	profileRuntime := siteRuntime.Profile()
	resourceType, exists := profileRuntime.Registry().ResourceType(
		item.Type,
	)
	if !exists {
		return Resource{}, fmt.Errorf(
			"resource references unknown type %q",
			item.Type,
		)
	}

	parent, err := s.relatedResource(
		ctx,
		item.ParentID,
		item.SiteID,
		known,
		"parent",
	)
	if err != nil {
		return Resource{}, err
	}

	payload := resourcetype.Payload{
		Template:         cloneTemplateCode(item.Template),
		ContentType:      cloneString(item.ContentType),
		Content:          item.Content,
		TargetResourceID: resourceTypeID(item.TargetResourceID),
		ExternalURL:      cloneString(item.ExternalURL),
		Settings:         cloneMap(item.Settings),
	}
	payload, err = resourceType.Normalize(payload)
	if err != nil {
		return Resource{}, fmt.Errorf(
			"normalize resource type %q: %w",
			item.Type,
			err,
		)
	}

	if payload.TargetResourceID != nil {
		targetID := ID(*payload.TargetResourceID)
		if targetID == item.ID && item.ID != 0 {
			return Resource{}, errors.New(
				"resource cannot target itself",
			)
		}
		if _, err := s.relatedResource(
			ctx,
			&targetID,
			item.SiteID,
			known,
			"target",
		); err != nil {
			return Resource{}, err
		}
	}

	if payload.Template == nil {
		if len(payload.Settings) != 0 {
			return Resource{}, errors.New(
				"resource without template has settings",
			)
		}
		payload.Settings = map[string]any{}
	} else {
		templateRuntime, exists := profileRuntime.Template(
			*payload.Template,
		)
		if !exists {
			return Resource{}, fmt.Errorf(
				"resource references unknown template %q",
				*payload.Template,
			)
		}

		settings, err := templateRuntime.FieldSchema().Validate(
			payload.Settings,
		)
		if err != nil {
			return Resource{}, fmt.Errorf(
				"validate resource template %q settings: %w",
				*payload.Template,
				err,
			)
		}
		payload.Settings = settings
	}

	switch resourceType.PathMode() {
	case resourcetype.PathRoute:
		item.Path, err = BuildPath(parent, item.Slug)
		if err != nil {
			return Resource{}, err
		}
	case resourcetype.PathNone:
		item.Path = nil
	default:
		return Resource{}, fmt.Errorf(
			"resource type %q has invalid path mode %q",
			item.Type,
			resourceType.PathMode(),
		)
	}

	item.Template = cloneTemplateCode(payload.Template)
	item.ContentType = cloneString(payload.ContentType)
	item.Content = payload.Content
	item.TargetResourceID = resourceID(payload.TargetResourceID)
	item.ExternalURL = cloneString(payload.ExternalURL)
	item.Settings = cloneMap(payload.Settings)
	return item, nil
}

func (s *Service) relatedResource(
	ctx context.Context,
	id *ID,
	siteID site.ID,
	known map[ID]Resource,
	role string,
) (*Resource, error) {
	if id == nil {
		return nil, nil
	}
	if *id <= 0 {
		return nil, fmt.Errorf("resource %s id is invalid", role)
	}

	var (
		item Resource
		err  error
	)
	if known != nil {
		var exists bool
		item, exists = known[*id]
		if !exists {
			return nil, fmt.Errorf(
				"resource %s %d not found",
				role,
				*id,
			)
		}
	} else {
		item, err = s.repository.ByID(ctx, *id)
		if err != nil {
			return nil, fmt.Errorf(
				"get resource %s %d: %w",
				role,
				*id,
				err,
			)
		}
	}
	if item.SiteID != siteID {
		return nil, fmt.Errorf(
			"resource %s %d belongs to another site",
			role,
			*id,
		)
	}

	item = Clone(item)
	return &item, nil
}

func (s *Service) ensureNoParentCycle(
	ctx context.Context,
	item Resource,
) error {
	if item.ParentID == nil {
		return nil
	}

	visited := map[ID]struct{}{item.ID: {}}
	currentID := cloneID(item.ParentID)
	for currentID != nil {
		if _, exists := visited[*currentID]; exists {
			return ErrInvalidTree
		}
		visited[*currentID] = struct{}{}

		current, err := s.repository.ByID(ctx, *currentID)
		if err != nil {
			return fmt.Errorf(
				"walk resource parent %d: %w",
				*currentID,
				err,
			)
		}
		if current.SiteID != item.SiteID {
			return errors.New("resource parent belongs to another site")
		}
		currentID = cloneID(current.ParentID)
	}

	return nil
}

func (s *Service) ensureNoRouteDescendants(
	ctx context.Context,
	item Resource,
	siteRuntime *site.Runtime,
) error {
	items, err := s.repository.ListBySite(ctx, item.SiteID)
	if err != nil {
		return fmt.Errorf(
			"list descendants of resource %d: %w",
			item.ID,
			err,
		)
	}

	byID := make(map[ID]Resource, len(items))
	for _, candidate := range items {
		byID[candidate.ID] = candidate
	}

	for _, candidate := range items {
		if candidate.ID == item.ID {
			continue
		}

		visited := make(map[ID]struct{})
		parentID := cloneID(candidate.ParentID)
		isDescendant := false
		for parentID != nil {
			if *parentID == item.ID {
				isDescendant = true
				break
			}
			if _, exists := visited[*parentID]; exists {
				return ErrInvalidTree
			}
			visited[*parentID] = struct{}{}

			parent, exists := byID[*parentID]
			if !exists {
				return fmt.Errorf(
					"resource %d references missing parent %d",
					candidate.ID,
					*parentID,
				)
			}
			parentID = cloneID(parent.ParentID)
		}
		if !isDescendant {
			continue
		}

		resourceType, exists := siteRuntime.Profile().
			Registry().
			ResourceType(candidate.Type)
		if !exists {
			return fmt.Errorf(
				"resource descendant %d references unknown type %q",
				candidate.ID,
				candidate.Type,
			)
		}
		if resourceType.PathMode() == resourcetype.PathRoute {
			return errors.New(
				"resource with route descendants cannot use no_path type",
			)
		}
	}

	return nil
}

func buildTree(items []Resource) ([]Node, error) {
	type mutableNode struct {
		resource Resource
		children []*mutableNode
	}

	nodes := make(map[ID]*mutableNode, len(items))
	for _, item := range items {
		if _, exists := nodes[item.ID]; exists {
			return nil, fmt.Errorf("duplicate resource id %d", item.ID)
		}
		nodes[item.ID] = &mutableNode{resource: Clone(item)}
	}

	roots := make([]*mutableNode, 0)
	for _, item := range items {
		node := nodes[item.ID]
		if item.ParentID == nil {
			roots = append(roots, node)
			continue
		}

		parent, exists := nodes[*item.ParentID]
		if !exists {
			return nil, fmt.Errorf(
				"resource %d references missing parent %d",
				item.ID,
				*item.ParentID,
			)
		}
		parent.children = append(parent.children, node)
	}

	sortNodes := func(nodes []*mutableNode) {
		sort.Slice(nodes, func(left, right int) bool {
			if nodes[left].resource.Sort != nodes[right].resource.Sort {
				return nodes[left].resource.Sort <
					nodes[right].resource.Sort
			}
			return nodes[left].resource.ID < nodes[right].resource.ID
		})
	}
	sortNodes(roots)
	for _, node := range nodes {
		sortNodes(node.children)
	}

	state := make(map[ID]uint8, len(nodes))
	visited := 0
	var convert func(*mutableNode) (Node, error)
	convert = func(current *mutableNode) (Node, error) {
		switch state[current.resource.ID] {
		case 1:
			return Node{}, ErrInvalidTree
		case 2:
			return Node{}, ErrInvalidTree
		}

		state[current.resource.ID] = 1
		result := Node{Resource: Clone(current.resource)}
		for _, child := range current.children {
			converted, err := convert(child)
			if err != nil {
				return Node{}, err
			}
			result.Children = append(result.Children, converted)
		}
		state[current.resource.ID] = 2
		visited++
		return result, nil
	}

	result := make([]Node, 0, len(roots))
	for _, root := range roots {
		converted, err := convert(root)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}
	if visited != len(nodes) {
		return nil, ErrInvalidTree
	}

	return result, nil
}

func validateContext(ctx context.Context, operation string) error {
	if ctx == nil {
		return fmt.Errorf("%s context is nil", operation)
	}
	return ctx.Err()
}

func boolDefault(value *bool, defaultValue bool) bool {
	if value == nil {
		return defaultValue
	}
	return *value
}

func resourceTypeID(value *ID) *int64 {
	if value == nil {
		return nil
	}
	result := int64(*value)
	return &result
}

func resourceID(value *int64) *ID {
	if value == nil {
		return nil
	}
	result := ID(*value)
	return &result
}

func validLookupPath(path string) bool {
	if path == "/" {
		return true
	}
	if !strings.HasPrefix(path, "/") ||
		strings.HasSuffix(path, "/") ||
		strings.Contains(path, "//") {
		return false
	}

	for _, segment := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		if !slugPattern.MatchString(segment) {
			return false
		}
	}
	return true
}

func equalStrings(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}
