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
var _ eventbus.Bus = extensionTestBus{}
var _ resourceextension.EditorProvider = (*extensionTestRuntime)(nil)
var _ SiteCatalog = extensionTestSites{}
var _ resource.ManagementRepository = (*extensionTestResources)(nil)
