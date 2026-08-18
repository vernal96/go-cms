package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/eventbus"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	corefile "github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type testAuthorizer struct{}

type testEventBus struct{}

func (testEventBus) Publish(context.Context, eventbus.Message) error {
	return nil
}

func (testEventBus) Consume(
	context.Context,
	eventbus.Subscription,
	eventbus.Handler,
) error {
	return nil
}

func (testAuthorizer) Check(
	context.Context,
	security.Actor,
	permission.Code,
) error {
	return nil
}

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

func (testModuleRuntime) Widgets() []widget.Widget {
	return []widget.Widget{testWidget{}}
}

type testWidget struct{}

func (testWidget) Definition() widget.Definition {
	required := true
	return widget.Definition{
		Code:        "summary",
		Label:       "Summary",
		Description: "Resource summary",
		Fields: []field.Definition{
			{
				Key:      "title",
				Type:     field.TypeString,
				Label:    "Title",
				Required: &required,
			},
			{
				Key:   "limit",
				Type:  field.TypeInteger,
				Label: "Limit",
			},
		},
	}
}

func (testWidget) New(
	values map[string]any,
) (widget.Instance, error) {
	return testWidgetInstance{values: values}, nil
}

type testWidgetInstance struct {
	values map[string]any
}

func (i testWidgetInstance) Render(
	context.Context,
	widget.RenderInput,
) (map[string]any, error) {
	return i.values, nil
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
	nextID       ID
	nextWidgetID widget.BindingID
	items        map[ID]Resource
	createError  error
	updateError  error
	deleteError  error
	deletedMedia []media.ID
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		nextID: 1, nextWidgetID: 1,
		items: make(map[ID]Resource),
	}
}

func (r *memoryRepository) CreateWidget(_ context.Context, id ID, binding widget.Binding) (widget.Binding, error) {
	item, exists := r.items[id]
	if !exists {
		return widget.Binding{}, ErrNotFound
	}
	binding = widget.CloneBinding(binding)
	binding.ID = r.nextWidgetID
	r.nextWidgetID++
	item.Widgets = append(item.Widgets, binding)
	r.items[id] = item
	return widget.CloneBinding(binding), nil
}

func (r *memoryRepository) UpdateWidget(_ context.Context, id ID, binding widget.Binding) (widget.Binding, error) {
	item, exists := r.items[id]
	if !exists {
		return widget.Binding{}, ErrNotFound
	}
	for index := range item.Widgets {
		if item.Widgets[index].ID == binding.ID {
			item.Widgets[index] = widget.CloneBinding(binding)
			r.items[id] = item
			return widget.CloneBinding(binding), nil
		}
	}
	return widget.Binding{}, ErrNotFound
}

func (r *memoryRepository) DeleteWidget(_ context.Context, id ID, bindingID widget.BindingID) error {
	item, exists := r.items[id]
	if !exists {
		return ErrNotFound
	}
	removed := false
	result := make([]widget.Binding, 0, len(item.Widgets))
	for _, binding := range item.Widgets {
		if binding.ID == bindingID {
			removed = true
			continue
		}
		result = append(result, binding)
	}
	if !removed {
		return ErrNotFound
	}
	positions := map[widget.AreaCode]int{widget.AreaBody: 0, widget.AreaSidebar: 0}
	for index := range result {
		result[index].Position = positions[result[index].Area]
		positions[result[index].Area]++
	}
	item.Widgets = result
	r.items[id] = item
	return nil
}

func (r *memoryRepository) ReorderWidgets(_ context.Context, id ID, order []widget.Order) ([]widget.Binding, error) {
	item, exists := r.items[id]
	if !exists {
		return nil, ErrNotFound
	}
	byID := make(map[widget.BindingID]widget.Binding, len(item.Widgets))
	for _, binding := range item.Widgets {
		byID[binding.ID] = binding
	}
	result := make([]widget.Binding, len(order))
	for index, ordered := range order {
		binding, exists := byID[ordered.ID]
		if !exists {
			return nil, ErrNotFound
		}
		binding.Area, binding.Position = ordered.Area, ordered.Position
		result[index] = binding
	}
	sort.SliceStable(result, func(left, right int) bool {
		if result[left].Area == result[right].Area {
			return result[left].Position < result[right].Position
		}
		return result[left].Area < result[right].Area
	})
	item.Widgets = result
	r.items[id] = item
	return widget.CloneBindings(result), nil
}

func (r *memoryRepository) Create(
	ctx context.Context,
	_ *security.UserID,
	item Resource,
	validate ValidateImageMedia,
) (Resource, error) {
	if r.createError != nil {
		return Resource{}, r.createError
	}
	if item.ImageMediaID != nil {
		if validate == nil {
			return Resource{}, errors.New("image media validator is nil")
		}
		for _, existing := range r.items {
			if existing.ImageMediaID != nil &&
				*existing.ImageMediaID == *item.ImageMediaID {
				return Resource{}, media.ErrAlreadyAttached
			}
		}
		if err := validate(ctx, *item.ImageMediaID); err != nil {
			return Resource{}, err
		}
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
	ctx context.Context,
	_ *security.UserID,
	expected Resource,
	item Resource,
	validate ValidateImageMedia,
) (Resource, error) {
	if r.updateError != nil {
		return Resource{}, r.updateError
	}

	current, exists := r.items[item.ID]
	if !exists {
		return Resource{}, ErrNotFound
	}
	if current.ImageMediaID == nil != (expected.ImageMediaID == nil) ||
		current.ImageMediaID != nil &&
			expected.ImageMediaID != nil &&
			*current.ImageMediaID != *expected.ImageMediaID {
		return Resource{}, ErrConflict
	}
	if item.ImageMediaID != nil {
		if validate == nil {
			return Resource{}, errors.New("image media validator is nil")
		}
		for id, existing := range r.items {
			if id != item.ID &&
				existing.ImageMediaID != nil &&
				*existing.ImageMediaID == *item.ImageMediaID {
				return Resource{}, media.ErrAlreadyAttached
			}
		}
		if err := validate(ctx, *item.ImageMediaID); err != nil {
			return Resource{}, err
		}
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
	if !sameTestMediaID(current.ImageMediaID, item.ImageMediaID) &&
		current.ImageMediaID != nil {
		r.deletedMedia = append(r.deletedMedia, *current.ImageMediaID)
	}
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
		if item := r.items[deletedID]; item.ImageMediaID != nil {
			r.deletedMedia = append(
				r.deletedMedia,
				*item.ImageMediaID,
			)
		}
		delete(r.items, deletedID)
	}
	return nil
}

func (r *memoryRepository) SoftDelete(
	_ context.Context,
	actorID *security.UserID,
	id ID,
) error {
	if r.deleteError != nil {
		return r.deleteError
	}
	if _, exists := r.items[id]; !exists {
		return ErrNotFound
	}
	now := time.Now().UTC()
	deleted := map[ID]bool{id: true}
	for changed := true; changed; {
		changed = false
		for candidateID, item := range r.items {
			if item.ParentID != nil && deleted[*item.ParentID] && !deleted[candidateID] {
				deleted[candidateID] = true
				changed = true
			}
		}
	}
	for deletedID := range deleted {
		item := r.items[deletedID]
		item.DeletedAt = &now
		item.DeletedBy = cloneUserID(actorID)
		r.items[deletedID] = item
	}
	return nil
}

func (r *memoryRepository) Restore(
	_ context.Context,
	_ *security.UserID,
	id ID,
	withDescendants bool,
) error {
	item, exists := r.items[id]
	if !exists {
		return ErrNotFound
	}
	if item.ParentID != nil && r.items[*item.ParentID].DeletedAt != nil {
		return ErrInvalidTree
	}
	restored := map[ID]bool{id: true}
	if withDescendants {
		for changed := true; changed; {
			changed = false
			for candidateID, candidate := range r.items {
				if candidate.ParentID != nil && restored[*candidate.ParentID] && !restored[candidateID] {
					restored[candidateID] = true
					changed = true
				}
			}
		}
	}
	for restoredID := range restored {
		candidate := r.items[restoredID]
		candidate.DeletedAt = nil
		candidate.DeletedBy = nil
		r.items[restoredID] = candidate
	}
	return nil
}

func sameTestMediaID(left, right *media.ID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

type testMediaService struct {
	items map[media.ID]media.ResolvedMedia
}

func newTestMediaService() *testMediaService {
	return &testMediaService{
		items: make(map[media.ID]media.ResolvedMedia),
	}
}

func (*testMediaService) Create(
	context.Context,
	security.Actor,
	media.CreateInput,
) (media.Media, error) {
	return media.Media{}, errors.New("not implemented")
}

func (s *testMediaService) Get(
	_ context.Context,
	_ security.Actor,
	id media.ID,
) (media.Media, error) {
	item, exists := s.items[id]
	if !exists {
		return media.Media{}, media.ErrNotFound
	}
	return media.Clone(item.Media), nil
}

func (s *testMediaService) Resolve(
	_ context.Context,
	_ security.Actor,
	id media.ID,
) (media.ResolvedMedia, error) {
	item, exists := s.items[id]
	if !exists {
		return media.ResolvedMedia{}, media.ErrNotFound
	}
	return media.ResolvedMedia{
		Media: media.Clone(item.Media),
		File:  corefile.Clone(item.File),
	}, nil
}

func (*testMediaService) Update(
	context.Context,
	security.Actor,
	media.UpdateInput,
) (media.Media, error) {
	return media.Media{}, errors.New("not implemented")
}

func (*testMediaService) Delete(
	context.Context,
	security.Actor,
	media.ID,
) error {
	return errors.New("not implemented")
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

func newTestService(
	t *testing.T,
) (*Service, *memoryRepository, *testMediaService) {
	t.Helper()

	required := true
	factory, err := kernel.NewProfileRuntimeFactory(
		testDatabaseResolver{},
		kernel.RuntimeServices{
			EventBus: testEventBus{},
			Logger:   slog.New(slog.NewJSONHandler(io.Discard, nil)),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	blueprint, err := factory.Compile(
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
					Layout: template.Layout{
						Body:    []template.LayoutItem{{Kind: template.ItemResourceSlot}},
						Sidebar: []template.LayoutItem{{Kind: template.ItemResourceSlot}},
					},
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
					Layout: template.Layout{
						Body:    []template.LayoutItem{{Kind: template.ItemResourceSlot}},
						Sidebar: []template.LayoutItem{{Kind: template.ItemResourceSlot}},
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	sites := make(testSites)
	for _, siteID := range []site.ID{1, 2} {
		runtime, err := site.NewRuntimeFromBlueprint(context.Background(), site.Site{
			ID:          siteID,
			ProfileCode: "test",
			Domain:      fmt.Sprintf("site-%d.example.com", siteID),
			Locale:      "en-US",
			Settings:    map[string]any{},
		}, blueprint)
		if err != nil {
			t.Fatal(err)
		}
		sites[siteID] = runtime
	}

	repository := newMemoryRepository()
	mediaService := newTestMediaService()
	service, err := NewService(
		repository,
		sites,
		mediaService,
		testAuthorizer{},
	)
	if err != nil {
		t.Fatal(err)
	}
	return service, repository, mediaService
}

func TestServiceCreatePageDefaultsAndTemplateSettings(t *testing.T) {
	service, _, _ := newTestService(t)
	templateCode := template.Code("article")

	home, err := service.Create(context.Background(), security.System(), CreateInput{
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
	about, err := service.Create(context.Background(), security.System(), CreateInput{
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

	_, err = service.Create(context.Background(), security.System(), CreateInput{
		SiteID: 1,
		Title:  "Missing template",
		Slug:   "missing-template",
	})
	if err == nil || !strings.Contains(err.Error(), "template is required") {
		t.Fatalf("missing template error = %v", err)
	}

	_, err = service.Create(context.Background(), security.System(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "Invalid settings",
		Slug:     "invalid-settings",
		Settings: map[string]any{"headline": "x"},
	})
	if err == nil || !strings.Contains(err.Error(), "min") {
		t.Fatalf("invalid settings error = %v", err)
	}

	_, err = service.Create(context.Background(), security.System(), CreateInput{
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

func TestServiceWidgetBindingsKeepIdentityAcrossReorderAndMove(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	templateCode := template.Code("empty")
	created, err := service.Create(ctx, security.System(), CreateInput{
		SiteID: 1, Template: &templateCode, Title: "Home",
	})
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	first, err := service.CreateWidget(ctx, security.System(), created.ID, CreateWidgetInput{
		Code: "test_summary", Area: widget.AreaBody, Columns: 12, Enabled: &enabled,
		Params: map[string]any{"title": "Primary", "limit": json.Number("3")},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateWidget(ctx, security.System(), created.ID, CreateWidgetInput{
		Code: "test_summary", Area: widget.AreaBody, Columns: 6, Enabled: &enabled,
		Params: map[string]any{"title": "Secondary"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID <= 0 || second.ID <= 0 || first.ID == second.ID || first.Params["limit"] != int64(3) {
		t.Fatalf("bindings = %#v / %#v", first, second)
	}
	updated, err := service.UpdateWidget(ctx, security.System(), created.ID, first.ID, UpdateWidgetInput{
		Columns: 8, Enabled: &enabled, Params: map[string]any{"title": "Updated"},
	})
	if err != nil || updated.ID != first.ID || updated.Params["title"] != "Updated" {
		t.Fatalf("updated = %#v, %v", updated, err)
	}
	ordered, err := service.ReorderWidgets(ctx, security.System(), created.ID, []widget.Order{
		{ID: second.ID, Area: widget.AreaBody, Position: 0},
		{ID: first.ID, Area: widget.AreaSidebar, Position: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ordered) != 2 || ordered[0].ID != second.ID || ordered[1].ID != first.ID || ordered[1].Area != widget.AreaSidebar {
		t.Fatalf("ordered = %#v", ordered)
	}
	stored, err := service.Get(ctx, security.System(), created.ID)
	if err != nil || stored.Widgets[1].ID != first.ID {
		t.Fatalf("stored = %#v, %v", stored.Widgets, err)
	}
	if err := service.DeleteWidget(ctx, security.System(), created.ID, second.ID); err != nil {
		t.Fatal(err)
	}
	stored, _ = service.Get(ctx, security.System(), created.ID)
	if len(stored.Widgets) != 1 || stored.Widgets[0].ID != first.ID || stored.Widgets[0].Position != 0 {
		t.Fatalf("after delete = %#v", stored.Widgets)
	}
}

func TestServiceRejectsInvalidWidgetMutations(t *testing.T) {
	service, _, _ := newTestService(t)
	ctx := context.Background()
	templateCode := template.Code("empty")
	created, err := service.Create(ctx, security.System(), CreateInput{SiteID: 1, Template: &templateCode, Title: "Home"})
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	tests := []struct {
		name  string
		input CreateWidgetInput
		match string
	}{
		{name: "unknown", input: CreateWidgetInput{Code: "missing_widget", Area: widget.AreaBody, Columns: 12, Enabled: &enabled, Params: map[string]any{}}, match: "unavailable"},
		{name: "required", input: CreateWidgetInput{Code: "test_summary", Area: widget.AreaBody, Columns: 12, Enabled: &enabled, Params: map[string]any{}}, match: "required"},
		{name: "area", input: CreateWidgetInput{Code: "test_summary", Area: "footer", Columns: 12, Enabled: &enabled, Params: map[string]any{"title": "Title"}}, match: "area"},
		{name: "columns", input: CreateWidgetInput{Code: "test_summary", Area: widget.AreaBody, Columns: 13, Enabled: &enabled, Params: map[string]any{"title": "Title"}}, match: "columns"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.CreateWidget(ctx, security.System(), created.ID, test.input)
			if err == nil || !strings.Contains(err.Error(), test.match) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestServiceBuiltInTypesAndNoPathType(t *testing.T) {
	service, _, _ := newTestService(t)
	templateCode := template.Code("empty")

	target, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "Target",
		Slug:     "target",
	})
	if err != nil {
		t.Fatal(err)
	}

	externalURL := "https://example.com/docs"
	link, err := service.Create(context.Background(), security.System(), CreateInput{
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
		security.System(),
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
		security.System(),
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
	_, err = service.Create(context.Background(), security.System(), CreateInput{
		SiteID:           1,
		Type:             resourcetype.ResourceLink,
		Title:            "Cross-site alias",
		Slug:             "cross-site",
		TargetResourceID: &otherSiteTarget.ID,
	})
	if err == nil || !strings.Contains(err.Error(), "another site") {
		t.Fatalf("cross-site target error = %v", err)
	}

	noPath, err := service.Create(context.Background(), security.System(), CreateInput{
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

	_, err = service.Create(context.Background(), security.System(), CreateInput{
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
	service, _, _ := newTestService(t)
	templateCode := template.Code("empty")

	first, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "First",
		Slug:     "first",
		Sort:     20,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "Second",
		Slug:     "second",
		Sort:     10,
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.Create(context.Background(), security.System(), CreateInput{
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
		security.System(),
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

	child, err = service.Update(context.Background(), security.System(), UpdateInput{
		ID:           child.ID,
		ParentID:     &second.ID,
		Type:         resourcetype.Page,
		Template:     &templateCode,
		ContentType:  testStringPointer("html"),
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
	grandchild, err = service.Get(context.Background(), security.System(), grandchild.ID)
	if err != nil {
		t.Fatal(err)
	}
	if grandchild.Path == nil ||
		*grandchild.Path != "/second/renamed/grandchild" {
		t.Fatalf("grandchild path = %#v", grandchild.Path)
	}

	tree, err := service.Tree(context.Background(), security.System(), 1)
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

	_, err = service.Update(context.Background(), security.System(), UpdateInput{
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
	service, _, _ := newTestService(t)
	article := template.Code("article")
	empty := template.Code("empty")

	page, err := service.Create(context.Background(), security.System(), CreateInput{
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
	child, err := service.Create(context.Background(), security.System(), CreateInput{
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
	page, err = service.Update(context.Background(), security.System(), UpdateInput{
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

	_, err = service.Update(context.Background(), security.System(), UpdateInput{
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

	child, err = service.Get(context.Background(), security.System(), child.ID)
	if err != nil {
		t.Fatal(err)
	}
	if child.Path == nil || *child.Path != "/page/child" {
		t.Fatalf("child changed after rejected update = %#v", child.Path)
	}
}

func TestServicePublicationAndRepositoryErrors(t *testing.T) {
	service, repository, _ := newTestService(t)
	templateCode := template.Code("empty")
	publishedAt := time.Now().UTC()
	unpublishedAt := publishedAt

	_, err := service.Create(context.Background(), security.System(), CreateInput{
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
	_, err = service.Create(context.Background(), security.System(), CreateInput{
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
	service, _, _ := newTestService(t)
	templateCode := template.Code("empty")

	root, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:   1,
		Template: &templateCode,
		Title:    "Root",
		Slug:     "root",
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:   1,
		ParentID: &root.ID,
		Template: &templateCode,
		Title:    "Child",
		Slug:     "child",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Create(context.Background(), security.System(), CreateInput{
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
		security.System(),
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

	if err := service.DeletePermanent(context.Background(), security.System(), root.ID); !errors.Is(
		err,
		ErrReferenced,
	) {
		t.Fatalf("referenced delete error = %v", err)
	}
	if err := service.DeletePermanent(
		context.Background(),
		security.System(),
		externalLink.ID,
	); err != nil {
		t.Fatal(err)
	}
	if err := service.DeletePermanent(context.Background(), security.System(), root.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(
		context.Background(),
		security.System(),
		child.ID,
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted child error = %v", err)
	}
}

func TestServiceSoftDeleteAndRestoreSubtree(t *testing.T) {
	service, repository, _ := newTestService(t)
	templateCode := template.Code("empty")
	root, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID: 1, Template: &templateCode, Title: "Root", Slug: "root",
	})
	if err != nil {
		t.Fatal(err)
	}
	child, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID: 1, ParentID: &root.ID, Template: &templateCode, Title: "Child", Slug: "child",
	})
	if err != nil {
		t.Fatal(err)
	}
	grandchild, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID: 1, ParentID: &child.ID, Template: &templateCode, Title: "Grandchild", Slug: "grandchild",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := service.Delete(context.Background(), security.System(), root.ID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []ID{root.ID, child.ID, grandchild.ID} {
		if repository.items[id].DeletedAt == nil {
			t.Fatalf("resource %d was not soft-deleted", id)
		}
	}
	if err := service.Restore(context.Background(), security.System(), child.ID, false); !errors.Is(err, ErrInvalidTree) {
		t.Fatalf("restore under deleted parent error = %v", err)
	}
	if err := service.Restore(context.Background(), security.System(), root.ID, false); err != nil {
		t.Fatal(err)
	}
	if repository.items[root.ID].DeletedAt != nil || repository.items[child.ID].DeletedAt == nil {
		t.Fatalf("single restore state = %#v", repository.items)
	}
	if err := service.Restore(context.Background(), security.System(), child.ID, true); err != nil {
		t.Fatal(err)
	}
	if repository.items[child.ID].DeletedAt != nil || repository.items[grandchild.ID].DeletedAt != nil {
		t.Fatalf("recursive restore state = %#v", repository.items)
	}
}

func TestServiceDetectsInvalidStoredResourcesOnRead(t *testing.T) {
	service, repository, _ := newTestService(t)
	templateCode := template.Code("article")
	item, err := service.Create(context.Background(), security.System(), CreateInput{
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

			_, err := service.Get(context.Background(), security.System(), item.ID)
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

func TestServiceResourceImageMediaValidationAndAttachment(
	t *testing.T,
) {
	service, _, mediaService := newTestService(t)
	addResolvedMedia(mediaService, 1, 101, "image/png")
	addResolvedMedia(mediaService, 2, 102, "application/pdf")
	templateCode := template.Code("article")

	imageMediaID := media.ID(1)
	root, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:       1,
		Template:     &templateCode,
		Title:        "Home",
		ImageMediaID: &imageMediaID,
		Settings:     map[string]any{"headline": "Home"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if root.ImageMediaID == nil || *root.ImageMediaID != imageMediaID {
		t.Fatalf("resource image media = %#v", root.ImageMediaID)
	}

	if _, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:       1,
		ParentID:     &root.ID,
		Template:     &templateCode,
		Title:        "Child",
		Slug:         "child",
		ImageMediaID: &imageMediaID,
		Settings:     map[string]any{"headline": "Child"},
	}); !errors.Is(err, media.ErrAlreadyAttached) {
		t.Fatalf("duplicate media attachment error = %v", err)
	}

	nonImageMediaID := media.ID(2)
	if _, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:       2,
		Template:     &templateCode,
		Title:        "Other home",
		ImageMediaID: &nonImageMediaID,
		Settings:     map[string]any{"headline": "Other"},
	}); !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("non-image media error = %v", err)
	}

	missingMediaID := media.ID(99)
	if _, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:       2,
		Template:     &templateCode,
		Title:        "Missing image",
		ImageMediaID: &missingMediaID,
		Settings:     map[string]any{"headline": "Missing"},
	}); !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("missing media error = %v", err)
	}
}

func TestServiceReplacingAndClearingImageDeletesOldMedia(
	t *testing.T,
) {
	service, repository, mediaService := newTestService(t)
	addResolvedMedia(mediaService, 1, 101, "image/png")
	addResolvedMedia(mediaService, 2, 102, "image/webp")
	templateCode := template.Code("article")

	firstMediaID := media.ID(1)
	item, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:       1,
		Template:     &templateCode,
		Title:        "Home",
		ImageMediaID: &firstMediaID,
		Settings:     map[string]any{"headline": "Home"},
	})
	if err != nil {
		t.Fatal(err)
	}

	secondMediaID := media.ID(2)
	input := updateInputFrom(item)
	input.ImageMediaID = &secondMediaID
	item, err = service.Update(context.Background(), security.System(), input)
	if err != nil {
		t.Fatal(err)
	}
	if item.ImageMediaID == nil || *item.ImageMediaID != secondMediaID {
		t.Fatalf("replaced image media = %#v", item.ImageMediaID)
	}
	if len(repository.deletedMedia) != 1 ||
		repository.deletedMedia[0] != firstMediaID {
		t.Fatalf(
			"deleted media after replacement = %#v",
			repository.deletedMedia,
		)
	}

	input = updateInputFrom(item)
	input.ImageMediaID = nil
	item, err = service.Update(context.Background(), security.System(), input)
	if err != nil {
		t.Fatal(err)
	}
	if item.ImageMediaID != nil {
		t.Fatalf("cleared image media = %#v", item.ImageMediaID)
	}
	if len(repository.deletedMedia) != 2 ||
		repository.deletedMedia[1] != secondMediaID {
		t.Fatalf(
			"deleted media after clear = %#v",
			repository.deletedMedia,
		)
	}
}

func TestServiceDeletingResourceTreeDeletesItsMedia(t *testing.T) {
	service, repository, mediaService := newTestService(t)
	addResolvedMedia(mediaService, 1, 101, "image/png")
	addResolvedMedia(mediaService, 2, 102, "image/jpeg")
	templateCode := template.Code("article")

	rootMediaID := media.ID(1)
	root, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:       1,
		Template:     &templateCode,
		Title:        "Home",
		ImageMediaID: &rootMediaID,
		Settings:     map[string]any{"headline": "Home"},
	})
	if err != nil {
		t.Fatal(err)
	}
	childMediaID := media.ID(2)
	if _, err := service.Create(context.Background(), security.System(), CreateInput{
		SiteID:       1,
		ParentID:     &root.ID,
		Template:     &templateCode,
		Title:        "Child",
		Slug:         "child",
		ImageMediaID: &childMediaID,
		Settings:     map[string]any{"headline": "Child"},
	}); err != nil {
		t.Fatal(err)
	}

	if err := service.DeletePermanent(context.Background(), security.System(), root.ID); err != nil {
		t.Fatal(err)
	}
	if len(repository.items) != 0 {
		t.Fatalf("resources after delete = %#v", repository.items)
	}

	deleted := make(map[media.ID]bool)
	for _, id := range repository.deletedMedia {
		deleted[id] = true
	}
	if !deleted[rootMediaID] || !deleted[childMediaID] {
		t.Fatalf("deleted media = %#v", repository.deletedMedia)
	}
}

func addResolvedMedia(
	service *testMediaService,
	id media.ID,
	fileID corefile.ID,
	mimeType string,
) {
	service.items[id] = media.ResolvedMedia{
		Media: media.Media{
			ID:     id,
			FileID: fileID,
			Params: map[string]any{},
		},
		File: corefile.File{
			ID:       fileID,
			MIMEType: mimeType,
		},
	}
}

func updateInputFrom(item Resource) UpdateInput {
	return UpdateInput{
		ID:               item.ID,
		ParentID:         cloneID(item.ParentID),
		Type:             item.Type,
		Template:         cloneTemplateCode(item.Template),
		ContentType:      cloneString(item.ContentType),
		Title:            item.Title,
		MenuTitle:        item.MenuTitle,
		Slug:             item.Slug,
		Content:          item.Content,
		ImageMediaID:     cloneMediaID(item.ImageMediaID),
		TargetResourceID: cloneID(item.TargetResourceID),
		ExternalURL:      cloneString(item.ExternalURL),
		IsPublic:         item.IsPublic,
		IsSearchable:     item.IsSearchable,
		InMenu:           item.InMenu,
		InSitemap:        item.InSitemap,
		Sort:             item.Sort,
		PublishedAt:      cloneTime(item.PublishedAt),
		UnpublishedAt:    cloneTime(item.UnpublishedAt),
		Settings:         cloneMap(item.Settings),
	}
}

func testStringPointer(value string) *string {
	return &value
}
