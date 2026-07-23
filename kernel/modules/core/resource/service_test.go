package resource

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
)

type testModule struct{}

func (testModule) Code() kernel.ModuleCode {
	return "test"
}

func (testModule) Registry() kernel.ModuleRegistry {
	return kernel.ModuleRegistry{
		FieldTypes: field.StandardTypes(),
		ResourceTypes: append(
			resourcetype.StandardTypes(),
			noPathType{},
		),
	}
}

func (testModule) Build(
	context.Context,
	kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	return testModuleRuntime{}, nil
}

type testModuleRuntime struct{}

func (testModuleRuntime) ModuleCode() kernel.ModuleCode {
	return "test"
}

type noPathType struct{}

func (noPathType) Code() resourcetype.Code {
	return "no_path"
}

func (noPathType) PathMode() resourcetype.PathMode {
	return resourcetype.PathNone
}

func (noPathType) Normalize(
	payload resourcetype.Payload,
) (resourcetype.Payload, error) {
	if payload.Template != nil ||
		payload.ContentType != nil ||
		payload.Content != "" ||
		payload.TargetResourceID != nil ||
		payload.ExternalURL != nil ||
		len(payload.Settings) != 0 {
		return resourcetype.Payload{}, errors.New(
			"no_path payload must be empty",
		)
	}
	return payload, nil
}

type testDatabaseResolver struct{}

func (testDatabaseResolver) MainModuleDatabase(
	kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	return nil, false
}

func (testDatabaseResolver) ModuleDatabase(
	kernel.ConnectionCode,
	kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	return nil, false
}

type testSites map[site.ID]*site.Runtime

func (s testSites) RuntimeByID(id site.ID) (*site.Runtime, bool) {
	runtime, exists := s[id]
	return runtime, exists
}

type memoryRepository struct {
	nextID      ID
	items       map[ID]Resource
	createError error
	updateError error
	deleteError error
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		nextID: 1,
		items:  make(map[ID]Resource),
	}
}

func (r *memoryRepository) Create(
	_ context.Context,
	item Resource,
) (Resource, error) {
	if r.createError != nil {
		return Resource{}, r.createError
	}

	item = Clone(item)
	item.ID = r.nextID
	r.nextID++
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now

	candidate := cloneResourceMap(r.items)
	candidate[item.ID] = item
	if err := validateMemoryUniqueness(candidate); err != nil {
		return Resource{}, err
	}
	r.items = candidate
	return Clone(item), nil
}

func (r *memoryRepository) ByID(
	_ context.Context,
	id ID,
) (Resource, error) {
	item, exists := r.items[id]
	if !exists {
		return Resource{}, ErrNotFound
	}
	return Clone(item), nil
}

func (r *memoryRepository) ByPath(
	_ context.Context,
	siteID site.ID,
	path string,
) (Resource, error) {
	for _, item := range r.items {
		if item.SiteID == siteID &&
			item.Path != nil &&
			*item.Path == path {
			return Clone(item), nil
		}
	}
	return Resource{}, ErrNotFound
}

func (r *memoryRepository) ListBySite(
	_ context.Context,
	siteID site.ID,
) ([]Resource, error) {
	result := make([]Resource, 0)
	for _, item := range r.items {
		if item.SiteID == siteID {
			result = append(result, Clone(item))
		}
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].ID < result[right].ID
	})
	return result, nil
}

func (r *memoryRepository) Update(
	_ context.Context,
	item Resource,
) (Resource, error) {
	if r.updateError != nil {
		return Resource{}, r.updateError
	}

	current, exists := r.items[item.ID]
	if !exists {
		return Resource{}, ErrNotFound
	}
	item = Clone(item)
	item.CreatedAt = current.CreatedAt
	item.UpdatedAt = time.Now().UTC()

	candidate := cloneResourceMap(r.items)
	candidate[item.ID] = item
	for iteration := 0; iteration <= len(candidate); iteration++ {
		changed := false
		for id, child := range candidate {
			if child.ParentID == nil || child.Path == nil {
				continue
			}
			parent, exists := candidate[*child.ParentID]
			if !exists {
				return Resource{}, ErrInvalidReference
			}
			path, err := BuildPath(&parent, child.Slug)
			if err != nil {
				return Resource{}, err
			}
			if !equalStrings(path, child.Path) {
				child.Path = path
				child.UpdatedAt = item.UpdatedAt
				candidate[id] = child
				changed = true
			}
		}
		if !changed {
			break
		}
		if iteration == len(candidate) {
			return Resource{}, ErrInvalidTree
		}
	}

	if err := validateMemoryUniqueness(candidate); err != nil {
		return Resource{}, err
	}
	r.items = candidate
	return Clone(r.items[item.ID]), nil
}

func (r *memoryRepository) Delete(
	_ context.Context,
	id ID,
) error {
	if r.deleteError != nil {
		return r.deleteError
	}
	if _, exists := r.items[id]; !exists {
		return ErrNotFound
	}

	deleted := map[ID]bool{id: true}
	for changed := true; changed; {
		changed = false
		for candidateID, item := range r.items {
			if item.ParentID != nil &&
				deleted[*item.ParentID] &&
				!deleted[candidateID] {
				deleted[candidateID] = true
				changed = true
			}
		}
	}
	for candidateID, item := range r.items {
		if deleted[candidateID] ||
			item.TargetResourceID == nil {
			continue
		}
		if deleted[*item.TargetResourceID] {
			return ErrReferenced
		}
	}
	for deletedID := range deleted {
		delete(r.items, deletedID)
	}
	return nil
}

func validateMemoryUniqueness(items map[ID]Resource) error {
	siblings := make(map[string]ID)
	paths := make(map[string]ID)
	for _, item := range items {
		parent := "root"
		if item.ParentID != nil {
			parent = fmt.Sprintf("%d", *item.ParentID)
		}
		siblingKey := fmt.Sprintf(
			"%d:%s:%s",
			item.SiteID,
			parent,
			item.Slug,
		)
		if _, exists := siblings[siblingKey]; exists {
			return ErrConflict
		}
		siblings[siblingKey] = item.ID

		if item.Path == nil {
			continue
		}
		pathKey := fmt.Sprintf("%d:%s", item.SiteID, *item.Path)
		if _, exists := paths[pathKey]; exists {
			return ErrConflict
		}
		paths[pathKey] = item.ID
	}
	return nil
}

func cloneResourceMap(source map[ID]Resource) map[ID]Resource {
	result := make(map[ID]Resource, len(source))
	for id, item := range source {
		result[id] = Clone(item)
	}
	return result
}

func newTestService(t *testing.T) (*Service, *memoryRepository) {
	t.Helper()

	required := true
	factory, err := kernel.NewProfileRuntimeFactory(testDatabaseResolver{})
	if err != nil {
		t.Fatal(err)
	}
	profileRuntime, err := factory.Make(
		context.Background(),
		kernel.Profile{
			Code: "test",
			Modules: []kernel.ProfileModule{{
				Module: testModule{},
			}},
			Templates: []template.Definition{
				{
					Code:  "article",
					Label: "Article",
					Fields: []field.Definition{{
						Key:      "headline",
						Type:     field.TypeString,
						Label:    "Headline",
						Required: &required,
						Rules:    []string{"min=2"},
					}},
				},
				{
					Code:  "empty",
					Label: "Empty",
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	sites := make(testSites)
	for _, siteID := range []site.ID{1, 2} {
		runtime, err := site.NewRuntime(site.Site{
			ID:          siteID,
			ProfileCode: "test",
			Domain:      fmt.Sprintf("site-%d.example.com", siteID),
			Locale:      "en-US",
			Settings:    map[string]any{},
		}, profileRuntime)
		if err != nil {
			t.Fatal(err)
		}
		sites[siteID] = runtime
	}

	repository := newMemoryRepository()
	service, err := NewService(repository, sites)
	if err != nil {
		t.Fatal(err)
	}
	return service, repository
}

func TestServiceCreatePageDefaultsAndTemplateSettings(t *testing.T) {
	service, _ := newTestService(t)
	templateCode := template.Code("article")

	home, err := service.Create(context.Background(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    " Home ",
		Settings: map[string]any{"headline": "Welcome"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if home.Type != resourcetype.Page ||
		home.Path == nil ||
		*home.Path != "/" ||
		home.ContentType == nil ||
		*home.ContentType != "html" ||
		home.Content != "" ||
		home.Title != "Home" {
		t.Fatalf("created homepage = %#v", home)
	}
	if !home.IsPublic ||
		!home.IsSearchable ||
		!home.InMenu ||
		!home.InSitemap {
		t.Fatalf("boolean defaults = %#v", home)
	}

	falseValue := false
	about, err := service.Create(context.Background(), CreateInput{
		SiteID:       1,
		ParentID:     &home.ID,
		Template:     &templateCode,
		Title:        "About",
		MenuTitle:    "",
		Slug:         "about",
		IsPublic:     &falseValue,
		IsSearchable: &falseValue,
		InMenu:       &falseValue,
		InSitemap:    &falseValue,
		Settings:     map[string]any{"headline": "About us"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if about.Path == nil || *about.Path != "/about" {
		t.Fatalf("about path = %#v", about.Path)
	}
	if about.IsPublic ||
		about.IsSearchable ||
		about.InMenu ||
		about.InSitemap ||
		about.MenuTitle != "" {
		t.Fatalf("explicit false values = %#v", about)
	}

	_, err = service.Create(context.Background(), CreateInput{
		SiteID: 1,
		Title:  "Missing template",
		Slug:   "missing-template",
	})
	if err == nil || !strings.Contains(err.Error(), "template is required") {
		t.Fatalf("missing template error = %v", err)
	}

	_, err = service.Create(context.Background(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "Invalid settings",
		Slug:     "invalid-settings",
		Settings: map[string]any{"headline": "x"},
	})
	if err == nil || !strings.Contains(err.Error(), "min") {
		t.Fatalf("invalid settings error = %v", err)
	}

	_, err = service.Create(context.Background(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "Invalid slug",
		Slug:     "Not-Valid",
		Settings: map[string]any{"headline": "Valid"},
	})
	if err == nil || !strings.Contains(err.Error(), "slug") {
		t.Fatalf("invalid slug error = %v", err)
	}
}

func TestServiceBuiltInTypesAndNoPathType(t *testing.T) {
	service, _ := newTestService(t)
	templateCode := template.Code("empty")

	target, err := service.Create(context.Background(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "Target",
		Slug:     "target",
	})
	if err != nil {
		t.Fatal(err)
	}

	externalURL := "https://example.com/docs"
	link, err := service.Create(context.Background(), CreateInput{
		SiteID:      1,
		Type:        resourcetype.Link,
		Title:       "External",
		Slug:        "external",
		ExternalURL: &externalURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if link.Path == nil || *link.Path != "/external" {
		t.Fatalf("link path = %#v", link.Path)
	}

	resourceLink, err := service.Create(
		context.Background(),
		CreateInput{
			SiteID:           1,
			Type:             resourcetype.ResourceLink,
			Title:            "Alias",
			Slug:             "alias",
			TargetResourceID: &target.ID,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if resourceLink.TargetResourceID == nil ||
		*resourceLink.TargetResourceID != target.ID {
		t.Fatalf("resource link = %#v", resourceLink)
	}

	otherSiteTarget, err := service.Create(
		context.Background(),
		CreateInput{
			SiteID:   2,
			Template: &templateCode,
			Title:    "Other site target",
			Slug:     "other",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Create(context.Background(), CreateInput{
		SiteID:           1,
		Type:             resourcetype.ResourceLink,
		Title:            "Cross-site alias",
		Slug:             "cross-site",
		TargetResourceID: &otherSiteTarget.ID,
	})
	if err == nil || !strings.Contains(err.Error(), "another site") {
		t.Fatalf("cross-site target error = %v", err)
	}

	noPath, err := service.Create(context.Background(), CreateInput{
		SiteID: 1,
		Type:   "no_path",
		Title:  "Container",
		Slug:   "container",
	})
	if err != nil {
		t.Fatal(err)
	}
	if noPath.Path != nil {
		t.Fatalf("no_path resource path = %#v", noPath.Path)
	}

	_, err = service.Create(context.Background(), CreateInput{
		SiteID:   1,
		ParentID: &noPath.ID,
		Template: &templateCode,
		Title:    "Route child",
		Slug:     "route-child",
	})
	if err == nil || !strings.Contains(err.Error(), "parent has no path") {
		t.Fatalf("route child error = %v", err)
	}
}

func TestServiceUpdateMovesSubtreeAndBuildsSortedTree(t *testing.T) {
	service, _ := newTestService(t)
	templateCode := template.Code("empty")

	first, err := service.Create(context.Background(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "First",
		Slug:     "first",
		Sort:     20,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(context.Background(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "Second",
		Slug:     "second",
		Sort:     10,
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.Create(context.Background(), CreateInput{
		SiteID:   1,
		ParentID: &first.ID,
		Template: &templateCode,
		Title:    "Child",
		Slug:     "child",
		Sort:     5,
	})
	if err != nil {
		t.Fatal(err)
	}
	grandchild, err := service.Create(
		context.Background(),
		CreateInput{
			SiteID:   1,
			ParentID: &child.ID,
			Template: &templateCode,
			Title:    "Grandchild",
			Slug:     "grandchild",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	child, err = service.Update(context.Background(), UpdateInput{
		ID:           child.ID,
		ParentID:     &second.ID,
		Type:         resourcetype.Page,
		Template:     &templateCode,
		ContentType:  testStringPointer("xml"),
		Title:        "Moved child",
		Slug:         "renamed",
		IsPublic:     true,
		IsSearchable: true,
		InMenu:       true,
		InSitemap:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if child.Path == nil || *child.Path != "/second/renamed" {
		t.Fatalf("moved child path = %#v", child.Path)
	}
	grandchild, err = service.Get(context.Background(), grandchild.ID)
	if err != nil {
		t.Fatal(err)
	}
	if grandchild.Path == nil ||
		*grandchild.Path != "/second/renamed/grandchild" {
		t.Fatalf("grandchild path = %#v", grandchild.Path)
	}

	tree, err := service.Tree(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 2 ||
		tree[0].Resource.ID != second.ID ||
		tree[1].Resource.ID != first.ID {
		t.Fatalf("root order = %#v", tree)
	}
	if len(tree[0].Children) != 1 ||
		tree[0].Children[0].Resource.ID != child.ID ||
		len(tree[0].Children[0].Children) != 1 {
		t.Fatalf("nested tree = %#v", tree)
	}

	_, err = service.Update(context.Background(), UpdateInput{
		ID:           second.ID,
		ParentID:     &grandchild.ID,
		Type:         resourcetype.Page,
		Template:     &templateCode,
		Title:        "Cycle",
		Slug:         "second",
		IsPublic:     true,
		IsSearchable: true,
		InMenu:       true,
		InSitemap:    true,
	})
	if !errors.Is(err, ErrInvalidTree) {
		t.Fatalf("cycle error = %v", err)
	}
}

func TestServiceUpdateFullyReplacesStateAndRejectsNoPathAncestor(
	t *testing.T,
) {
	service, _ := newTestService(t)
	article := template.Code("article")
	empty := template.Code("empty")

	page, err := service.Create(context.Background(), CreateInput{
		SiteID:   1,
		Template: &article,
		Title:    "Page",
		Slug:     "page",
		Content:  "old",
		Settings: map[string]any{"headline": "Old title"},
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.Create(context.Background(), CreateInput{
		SiteID:   1,
		ParentID: &page.ID,
		Template: &empty,
		Title:    "Child",
		Slug:     "child",
	})
	if err != nil {
		t.Fatal(err)
	}

	externalURL := "/new-target"
	page, err = service.Update(context.Background(), UpdateInput{
		ID:           page.ID,
		Type:         resourcetype.Link,
		Title:        "Link now",
		Slug:         "page",
		ExternalURL:  &externalURL,
		IsPublic:     false,
		IsSearchable: false,
		InMenu:       false,
		InSitemap:    false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if page.Template != nil ||
		page.ContentType != nil ||
		page.Content != "" ||
		len(page.Settings) != 0 ||
		page.IsPublic ||
		page.IsSearchable ||
		page.InMenu ||
		page.InSitemap {
		t.Fatalf("fully replaced page = %#v", page)
	}

	_, err = service.Update(context.Background(), UpdateInput{
		ID:           page.ID,
		Type:         "no_path",
		Title:        "Container",
		Slug:         "page",
		IsPublic:     true,
		IsSearchable: true,
		InMenu:       true,
		InSitemap:    true,
	})
	if err == nil ||
		!strings.Contains(err.Error(), "route descendants") {
		t.Fatalf("no_path ancestor error = %v", err)
	}

	child, err = service.Get(context.Background(), child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if child.Path == nil || *child.Path != "/page/child" {
		t.Fatalf("child changed after rejected update = %#v", child.Path)
	}
}

func TestServicePublicationAndRepositoryErrors(t *testing.T) {
	service, repository := newTestService(t)
	templateCode := template.Code("empty")
	publishedAt := time.Now().UTC()
	unpublishedAt := publishedAt

	_, err := service.Create(context.Background(), CreateInput{
		SiteID:        1,
		Template:      &templateCode,
		Title:         "Dates",
		Slug:          "dates",
		PublishedAt:   &publishedAt,
		UnpublishedAt: &unpublishedAt,
	})
	if err == nil || !strings.Contains(err.Error(), "after") {
		t.Fatalf("publication error = %v", err)
	}

	repository.createError = errors.New("storage unavailable")
	_, err = service.Create(context.Background(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "Repository error",
		Slug:     "repository-error",
	})
	if err == nil || !strings.Contains(err.Error(), "storage unavailable") {
		t.Fatalf("repository error = %v", err)
	}
}

func TestServiceDeleteCascadeAndReferenceProtection(t *testing.T) {
	service, _ := newTestService(t)
	templateCode := template.Code("empty")

	root, err := service.Create(context.Background(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "Root",
		Slug:     "root",
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.Create(context.Background(), CreateInput{
		SiteID:   1,
		ParentID: &root.ID,
		Template: &templateCode,
		Title:    "Child",
		Slug:     "child",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Create(context.Background(), CreateInput{
		SiteID:           1,
		ParentID:         &root.ID,
		Type:             resourcetype.ResourceLink,
		Title:            "Internal link",
		Slug:             "internal-link",
		TargetResourceID: &child.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	externalLink, err := service.Create(
		context.Background(),
		CreateInput{
			SiteID:           1,
			Type:             resourcetype.ResourceLink,
			Title:            "External link",
			Slug:             "external-link",
			TargetResourceID: &child.ID,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Delete(context.Background(), root.ID); !errors.Is(
		err,
		ErrReferenced,
	) {
		t.Fatalf("referenced delete error = %v", err)
	}
	if err := service.Delete(
		context.Background(),
		externalLink.ID,
	); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(context.Background(), root.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(
		context.Background(),
		child.ID,
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted child error = %v", err)
	}
}

func TestServiceDetectsInvalidStoredResourcesOnRead(t *testing.T) {
	service, repository := newTestService(t)
	templateCode := template.Code("article")
	item, err := service.Create(context.Background(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "Stored",
		Slug:     "stored",
		Settings: map[string]any{"headline": "Stored headline"},
	})
	if err != nil {
		t.Fatal(err)
	}

	original := Clone(repository.items[item.ID])
	testCases := []struct {
		name     string
		mutate   func(*Resource)
		contains string
	}{
		{
			name: "unknown type",
			mutate: func(item *Resource) {
				item.Type = "removed"
			},
			contains: "unknown type",
		},
		{
			name: "unknown template",
			mutate: func(item *Resource) {
				value := template.Code("removed")
				item.Template = &value
			},
			contains: "unknown template",
		},
		{
			name: "invalid settings",
			mutate: func(item *Resource) {
				item.Settings = map[string]any{"headline": "x"}
			},
			contains: "min",
		},
		{
			name: "inconsistent path",
			mutate: func(item *Resource) {
				value := "/wrong"
				item.Path = &value
			},
			contains: "path is inconsistent",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			corrupted := Clone(original)
			testCase.mutate(&corrupted)
			repository.items[item.ID] = corrupted

			_, err := service.Get(context.Background(), item.ID)
			if err == nil || !strings.Contains(
				err.Error(),
				testCase.contains,
			) {
				t.Fatalf("get error = %v", err)
			}

			repository.items[item.ID] = Clone(original)
		})
	}
}

func testStringPointer(value string) *string {
	return &value
}
