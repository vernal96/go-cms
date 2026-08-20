package admin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/eventbus"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	corewidgets "github.com/vernal96/go-cms/kernel/modules/core/widgets"
	"github.com/vernal96/go-cms/kernel/modules/resourceextension"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type extensionTestDatabaseResolver struct{}

func (extensionTestDatabaseResolver) MainModuleDatabase(
	kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	return nil, false
}
func (extensionTestDatabaseResolver) ModuleDatabase(
	kernel.ConnectionCode,
	kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	return nil, false
}

type extensionTestBus struct{}

func (extensionTestBus) Publish(context.Context, eventbus.Message) error { return nil }
func (extensionTestBus) Consume(
	context.Context,
	eventbus.Subscription,
	eventbus.Handler,
) error {
	return nil
}

type extensionTestModule struct{ editor resourceextension.Editor }

func (extensionTestModule) Code() kernel.ModuleCode { return "feature" }
func (m extensionTestModule) Build(
	context.Context,
	kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	return &extensionTestRuntime{editor: m.editor}, nil
}

type extensionTestRuntime struct{ editor resourceextension.Editor }

func (*extensionTestRuntime) ModuleCode() kernel.ModuleCode { return "feature" }
func (r *extensionTestRuntime) ResourceEditorExtension() resourceextension.Editor {
	return r.editor
}

type widgetMetadataModule struct{}

func (widgetMetadataModule) Code() kernel.ModuleCode { return "feature" }
func (widgetMetadataModule) Registry() kernel.ModuleRegistry {
	return kernel.ModuleRegistry{ResourceTypes: resourcetype.StandardTypes()}
}
func (widgetMetadataModule) ModuleDescriptor() kernel.ModuleDescriptor {
	return kernel.ModuleDescriptor{Label: "Feature widgets", Description: "Feature description"}
}
func (widgetMetadataModule) Build(context.Context, kernel.ModuleContext) (kernel.ModuleRuntime, error) {
	return widgetMetadataRuntime{}, nil
}

type widgetMetadataRuntime struct{}

func (widgetMetadataRuntime) ModuleCode() kernel.ModuleCode { return "feature" }
func (widgetMetadataRuntime) Widgets() []widget.Widget      { return corewidgets.All()[:1] }

type extensionTestEditor struct {
	saved bool
}

func (*extensionTestEditor) Metadata() resourceextension.Metadata {
	return resourceextension.Metadata{
		Code: "test", Title: "Test",
		AppliesTo: []resourcetype.Code{resourcetype.Page},
	}
}
func (*extensionTestEditor) AppliesTo(code resourcetype.Code) bool {
	return code == resourcetype.Page
}
func (*extensionTestEditor) Read(
	context.Context,
	resourceextension.Request,
) (any, error) {
	return map[string]any{"value": "loaded"}, nil
}
func (e *extensionTestEditor) Save(
	_ context.Context,
	_ resourceextension.Request,
	_ json.RawMessage,
) (any, error) {
	e.saved = true
	return map[string]any{"value": "saved"}, nil
}
func (*extensionTestEditor) Preview(
	context.Context,
	resourceextension.Request,
	json.RawMessage,
) (any, error) {
	return map[string]any{"value": "preview"}, nil
}

type extensionTestSites struct{ runtime *site.Runtime }

func (s extensionTestSites) RuntimeByID(id site.ID) (*site.Runtime, bool) {
	return s.runtime, s.runtime != nil && s.runtime.Site().ID == id
}
func (extensionTestSites) Create(
	context.Context,
	security.Actor,
	site.CreateInput,
) (*site.Runtime, error) {
	return nil, errors.New("not implemented")
}
func (extensionTestSites) Update(
	context.Context,
	security.Actor,
	site.UpdateInput,
) (*site.Runtime, error) {
	return nil, errors.New("not implemented")
}
func (extensionTestSites) Delete(context.Context, security.Actor, site.ID) error {
	return errors.New("not implemented")
}

type extensionTestResources struct{ item resource.Resource }

func (r *extensionTestResources) Create(
	context.Context,
	*security.UserID,
	resource.Resource,
	resource.ValidateImageMedia,
) (resource.Resource, error) {
	return resource.Resource{}, errors.New("not implemented")
}
func (r *extensionTestResources) ByID(
	context.Context,
	resource.ID,
) (resource.Resource, error) {
	return r.item, nil
}
func (*extensionTestResources) ByPath(
	context.Context,
	site.ID,
	string,
) (resource.Resource, error) {
	return resource.Resource{}, errors.New("not implemented")
}
func (*extensionTestResources) ListBySite(
	context.Context,
	site.ID,
) ([]resource.Resource, error) {
	return nil, errors.New("not implemented")
}
func (*extensionTestResources) Update(
	context.Context,
	*security.UserID,
	resource.Resource,
	resource.Resource,
	resource.ValidateImageMedia,
) (resource.Resource, error) {
	return resource.Resource{}, errors.New("not implemented")
}
func (*extensionTestResources) Delete(context.Context, resource.ID) error {
	return errors.New("not implemented")
}
func (r *extensionTestResources) CreateWidget(
	_ context.Context,
	_ resource.ID,
	binding widget.Binding,
) (widget.Binding, error) {
	binding.ID = widget.BindingID(len(r.item.Widgets) + 1)
	r.item.Widgets = append(r.item.Widgets, widget.CloneBinding(binding))
	return widget.CloneBinding(binding), nil
}
func (r *extensionTestResources) UpdateWidget(
	_ context.Context,
	_ resource.ID,
	binding widget.Binding,
) (widget.Binding, error) {
	for index := range r.item.Widgets {
		if r.item.Widgets[index].ID == binding.ID {
			r.item.Widgets[index] = widget.CloneBinding(binding)
			return widget.CloneBinding(binding), nil
		}
	}
	return widget.Binding{}, resource.ErrNotFound
}
func (r *extensionTestResources) DeleteWidget(
	_ context.Context,
	_ resource.ID,
	bindingID widget.BindingID,
) error {
	for index, binding := range r.item.Widgets {
		if binding.ID != bindingID {
			continue
		}
		area := binding.Area
		position := binding.Position
		r.item.Widgets = append(r.item.Widgets[:index], r.item.Widgets[index+1:]...)
		for itemIndex := range r.item.Widgets {
			if r.item.Widgets[itemIndex].Area == area && r.item.Widgets[itemIndex].Position > position {
				r.item.Widgets[itemIndex].Position--
			}
		}
		return nil
	}
	return resource.ErrNotFound
}
func (r *extensionTestResources) ReorderWidgets(
	_ context.Context,
	_ resource.ID,
	order []widget.Order,
) ([]widget.Binding, error) {
	byID := make(map[widget.BindingID]widget.Binding, len(r.item.Widgets))
	for _, binding := range r.item.Widgets {
		byID[binding.ID] = binding
	}
	result := make([]widget.Binding, 0, len(order))
	for _, item := range order {
		binding, exists := byID[item.ID]
		if !exists {
			return nil, resource.ErrNotFound
		}
		binding.Area = item.Area
		binding.Position = item.Position
		result = append(result, binding)
	}
	r.item.Widgets = widget.CloneBindings(result)
	return widget.CloneBindings(result), nil
}
func (r *extensionTestResources) ExistsInSite(
	_ context.Context,
	siteID site.ID,
	resourceID resource.ID,
) (bool, error) {
	return r.item.SiteID == siteID && r.item.ID == resourceID, nil
}
func (*extensionTestResources) ListChildren(
	context.Context,
	site.ID,
	*resource.ID,
) ([]resource.Child, error) {
	return nil, errors.New("not implemented")
}

var _ resource.WidgetRepository = (*extensionTestResources)(nil)

type extensionTestPolicy struct{ err error }

func (p extensionTestPolicy) Scope(
	context.Context,
	security.Actor,
) (SiteAccessScope, error) {
	return SiteAccessScope{All: true}, p.err
}
func (p extensionTestPolicy) Check(
	context.Context,
	security.Actor,
	site.ID,
) error {
	return p.err
}

func extensionManagement(
	t *testing.T,
	editor resourceextension.Editor,
	resourceType resourcetype.Code,
) (*Management, *extensionTestEditor) {
	t.Helper()
	var modules []kernel.ProfileModule
	if editor != nil {
		modules = []kernel.ProfileModule{{
			Module: extensionTestModule{editor: editor},
		}}
	}
	profile := kernel.Profile{
		Code: "test", Modules: modules,
	}
	factory, err := kernel.NewProfileRuntimeFactory(
		extensionTestDatabaseResolver{},
		kernel.RuntimeServices{
			Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
			EventBus: extensionTestBus{},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	blueprint, err := factory.Compile(context.Background(), profile)
	if err != nil {
		t.Fatal(err)
	}
	siteRuntime, err := site.NewRuntimeFromBlueprint(context.Background(), site.Site{
		ID: 7, ProfileCode: "test", Domain: "example.com", Locale: "ru-RU",
		Settings: map[string]any{},
	}, blueprint)
	if err != nil {
		t.Fatal(err)
	}
	testEditor, _ := editor.(*extensionTestEditor)
	resources := &extensionTestResources{item: resource.Resource{
		ID: 9, SiteID: 7, Type: resourceType,
	}}
	return &Management{
		sites:        extensionTestSites{runtime: siteRuntime},
		resourceRepo: resources,
		authorizer:   managementAuthorizer{denied: map[permission.Code]error{}},
		policy:       extensionTestPolicy{},
	}, testEditor
}

func TestResourceMetadataHasNoExtensionsWithoutProfileProvider(t *testing.T) {
	t.Parallel()
	management, _ := extensionManagement(t, nil, resourcetype.Page)
	metadata, err := management.ResourceMetadata(
		context.Background(), security.User(1), 7,
	)
	if err != nil || len(metadata.Extensions) != 0 {
		t.Fatalf("metadata = %#v, %v", metadata, err)
	}
}

func TestResourceMetadataDescribesTemplateSlotsAndProfileWidgets(t *testing.T) {
	compact := widget.NewView(corewidgets.Content, "compact", "Compact")
	profile := kernel.Profile{
		Code:    "widgets",
		Modules: []kernel.ProfileModule{{Module: widgetMetadataModule{}}},
		Templates: []template.Definition{{
			Code: "page", Label: "Page",
			Layout: template.Layout{
				Body: []template.Item{
					template.Widget{Widget: corewidgets.Content},
					template.ResourceWidgets{},
				},
				Sidebar: []template.Item{template.ResourceWidgets{}},
			},
		}},
		WidgetViews: []widget.View{compact},
	}
	factory, err := kernel.NewProfileRuntimeFactory(extensionTestDatabaseResolver{}, kernel.RuntimeServices{
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), EventBus: extensionTestBus{},
	})
	if err != nil {
		t.Fatal(err)
	}
	blueprint, err := factory.Compile(context.Background(), profile)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := site.NewRuntimeFromBlueprint(context.Background(), site.Site{
		ID: 7, ProfileCode: profile.Code, Domain: "example.com", Locale: "ru-RU", Settings: map[string]any{},
	}, blueprint)
	if err != nil {
		t.Fatal(err)
	}
	management := &Management{
		sites:      extensionTestSites{runtime: runtime},
		authorizer: managementAuthorizer{denied: map[permission.Code]error{}},
		policy:     extensionTestPolicy{},
	}
	metadata, err := management.ResourceMetadata(context.Background(), security.User(1), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(metadata.Templates) != 1 || !metadata.Templates[0].SupportsResourceWidgets ||
		len(metadata.Templates[0].WidgetAreas) != 2 {
		t.Fatalf("templates = %#v", metadata.Templates)
	}
	if len(metadata.Widgets) != 1 || metadata.Widgets[0].Code != "feature_content" ||
		metadata.Widgets[0].ModuleCode != "feature" || metadata.Widgets[0].ModuleLabel != "Feature widgets" ||
		metadata.Widgets[0].SummaryFields == nil ||
		len(metadata.Widgets[0].Views) != 1 || metadata.Widgets[0].Views[0].Code != "compact" {
		t.Fatalf("widgets = %#v", metadata.Widgets)
	}
}

func TestResourceExtensionMetadataAndOperationsAreProfileScoped(t *testing.T) {
	t.Parallel()
	management, editor := extensionManagement(
		t,
		&extensionTestEditor{},
		resourcetype.Page,
	)
	metadata, err := management.ResourceMetadata(
		context.Background(), security.User(1), 7,
	)
	if err != nil || len(metadata.Extensions) != 1 ||
		metadata.Extensions[0].Code != "test" {
		t.Fatalf("metadata = %#v, %v", metadata, err)
	}
	if _, err := management.ResourceExtension(
		context.Background(), security.User(1), 7, 9, "test",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := management.SaveResourceExtension(
		context.Background(), security.User(1), 7, 9, "test",
		json.RawMessage(`{}`),
	); err != nil || !editor.saved {
		t.Fatalf("save = %v, %t", err, editor.saved)
	}
	if _, err := management.PreviewResourceExtension(
		context.Background(), security.User(1), 7, 9, "test",
		json.RawMessage(`{}`),
	); err != nil {
		t.Fatal(err)
	}
}

func TestResourceExtensionChecksPermissionSiteAndApplicability(t *testing.T) {
	t.Parallel()
	management, _ := extensionManagement(
		t,
		&extensionTestEditor{},
		resourcetype.Page,
	)
	management.authorizer = managementAuthorizer{denied: map[permission.Code]error{
		ResourceUpdatePermission: security.ErrForbidden,
	}}
	if _, err := management.SaveResourceExtension(
		context.Background(), security.User(1), 7, 9, "test",
		json.RawMessage(`{}`),
	); !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("update permission error = %v", err)
	}
	management.authorizer = managementAuthorizer{denied: map[permission.Code]error{}}
	management.policy = extensionTestPolicy{err: security.ErrForbidden}
	if _, err := management.ResourceExtension(
		context.Background(), security.User(1), 7, 9, "test",
	); !errors.Is(err, security.ErrForbidden) {
		t.Fatalf("site access error = %v", err)
	}
	management.policy = extensionTestPolicy{}
	management.resourceRepo.(*extensionTestResources).item.SiteID = 8
	if _, err := management.ResourceExtension(
		context.Background(), security.User(1), 7, 9, "test",
	); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("site scope error = %v", err)
	}

	linkManagement, _ := extensionManagement(
		t,
		&extensionTestEditor{},
		resourcetype.Link,
	)
	if _, err := linkManagement.ResourceExtension(
		context.Background(), security.User(1), 7, 9, "test",
	); !errors.Is(err, resource.ErrNotFound) {
		t.Fatalf("link applicability error = %v", err)
	}
}

var _ kernel.Module = extensionTestModule{}
var _ kernel.Module = widgetMetadataModule{}
var _ kernel.ModuleDescriptorProvider = widgetMetadataModule{}
var _ widget.Provider = widgetMetadataRuntime{}
var _ eventbus.Bus = extensionTestBus{}
var _ resourceextension.EditorProvider = (*extensionTestRuntime)(nil)
var _ SiteCatalog = extensionTestSites{}
var _ resource.ManagementRepository = (*extensionTestResources)(nil)
