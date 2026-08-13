package kernel_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/cache"
	"github.com/vernal96/go-cms/kernel/eventbus"
	"github.com/vernal96/go-cms/kernel/filesystem"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	corefile "github.com/vernal96/go-cms/kernel/modules/core/file"
	coregroup "github.com/vernal96/go-cms/kernel/modules/core/group"
	coremedia "github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	coreuser "github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/modules/core/widget"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type emptyDatabaseResolver struct{}

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

func testRuntimeServices() kernel.RuntimeServices {
	return kernel.RuntimeServices{
		EventBus: testEventBus{},
		Logger:   slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}
}

func TestProfileRuntimeListsModuleRuntimesInProfileOrder(t *testing.T) {
	t.Parallel()
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		testRuntimeServices(),
	)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := factory.Make(context.Background(), kernel.Profile{
		Code: "ordered",
		Modules: []kernel.ProfileModule{
			{Module: registryModule{code: "first"}},
			{Module: registryModule{code: "second"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	modules := runtime.Modules()
	if len(modules) != 2 || modules[0].ModuleCode() != "first" ||
		modules[1].ModuleCode() != "second" {
		t.Fatalf("module runtimes = %#v", modules)
	}
	modules[0] = nil
	if runtime.Modules()[0] == nil {
		t.Fatal("module runtime slice shares caller memory")
	}
}

func (emptyDatabaseResolver) MainModuleDatabase(
	kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	return nil, false
}

func (emptyDatabaseResolver) ModuleDatabase(
	kernel.ConnectionCode,
	kernel.ModuleCode,
) (kernel.ModuleDatabase, bool) {
	return nil, false
}

type registryModule struct {
	code               kernel.ModuleCode
	fieldTypes         []field.Type
	resourceTypes      []resourcetype.Type
	permissionEntities []permission.Entity
	expectType         field.TypeCode
	expectResourceType resourcetype.Code
	expectPermission   permission.Code
}

func (m registryModule) Code() kernel.ModuleCode {
	return m.code
}

func (m registryModule) Registry() kernel.ModuleRegistry {
	return kernel.ModuleRegistry{
		FieldTypes: append([]field.Type(nil), m.fieldTypes...),
		ResourceTypes: append(
			[]resourcetype.Type(nil),
			m.resourceTypes...,
		),
		PermissionEntities: append(
			[]permission.Entity(nil),
			m.permissionEntities...,
		),
	}
}

func (m registryModule) Build(
	_ context.Context,
	ctx kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	if m.expectType != "" {
		if _, exists := ctx.Registry().FieldType(m.expectType); !exists {
			return nil, errors.New("expected field type is not registered")
		}
	}
	if m.expectResourceType != "" {
		if _, exists := ctx.Registry().ResourceType(
			m.expectResourceType,
		); !exists {
			return nil, errors.New(
				"expected resource type is not registered",
			)
		}
	}
	if m.expectPermission != "" {
		if _, exists := ctx.Registry().Permission(
			m.expectPermission,
		); !exists {
			return nil, errors.New(
				"expected permission is not registered",
			)
		}
	}

	return registryRuntime{code: m.code}, nil
}

type registryRuntime struct {
	code kernel.ModuleCode
}

type widgetProviderModule struct {
	code    kernel.ModuleCode
	widgets []widget.Widget
}

func (m widgetProviderModule) Code() kernel.ModuleCode {
	return m.code
}

func (m widgetProviderModule) Build(
	context.Context,
	kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	return widgetProviderRuntime{
		code:    m.code,
		widgets: append([]widget.Widget(nil), m.widgets...),
	}, nil
}

type widgetProviderRuntime struct {
	code    kernel.ModuleCode
	widgets []widget.Widget
}

func (r widgetProviderRuntime) ModuleCode() kernel.ModuleCode {
	return r.code
}

func (r widgetProviderRuntime) Widgets() []widget.Widget {
	return append([]widget.Widget(nil), r.widgets...)
}

type runtimeWidget struct {
	definition widget.Definition
}

func (w runtimeWidget) Definition() widget.Definition {
	return widget.CloneDefinition(w.definition)
}

func (runtimeWidget) New(
	values map[string]any,
) (widget.Instance, error) {
	return runtimeWidgetInstance{values: values}, nil
}

type runtimeWidgetInstance struct {
	values map[string]any
}

func (i runtimeWidgetInstance) Render(
	context.Context,
	widget.RenderInput,
) (map[string]any, error) {
	return i.values, nil
}

type markerFileService struct {
	corefile.Service
}

type markerMediaService struct {
	coremedia.Service
}

type markerUserService struct {
	coreuser.Service
}

type markerGroupService struct {
	coregroup.Service
}

type markerAuthorizer struct{}

func (*markerAuthorizer) Check(
	context.Context,
	security.Actor,
	permission.Code,
) error {
	return nil
}

type fileAwareModule struct {
	expectedFiles         corefile.Service
	expectedMedia         coremedia.Service
	expectedUsers         coreuser.Service
	expectedGroups        coregroup.Service
	expectedAuthorization security.Authorizer
}

type cacheAwareModule struct{}

type infrastructureAwareModule struct {
	eventBus eventbus.Bus
	disk     filesystem.Disk
}

type loggerAwareModule struct {
	code kernel.ModuleCode
}

func (*infrastructureAwareModule) Code() kernel.ModuleCode {
	return "infrastructure-aware"
}

func (m *infrastructureAwareModule) Build(
	_ context.Context,
	ctx kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	if ctx.EventBus() != m.eventBus {
		return nil, errors.New("module event bus does not match")
	}
	if ctx.Filesystems() == nil {
		return nil, errors.New("module filesystem manager is nil")
	}
	disk, exists := ctx.Filesystems().Disk("assets")
	if !exists || disk != m.disk {
		return nil, errors.New("module filesystem alias is missing")
	}
	if _, exists := ctx.Filesystems().Disk("private"); exists {
		return nil, errors.New("module resolved unbound filesystem")
	}
	return registryRuntime{code: m.Code()}, nil
}

func (m loggerAwareModule) Code() kernel.ModuleCode {
	return m.code
}

func (m loggerAwareModule) Build(
	_ context.Context,
	ctx kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	if ctx.Logger() == nil {
		return nil, errors.New("module logger is nil")
	}
	ctx.Logger().Info("module log")
	return registryRuntime{code: m.code}, nil
}

func (*cacheAwareModule) Code() kernel.ModuleCode {
	return "cache-aware"
}

func (m *cacheAwareModule) Build(
	_ context.Context,
	ctx kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	if ctx.Caches() == nil {
		return nil, errors.New("module cache manager is nil")
	}
	fast, fastExists := ctx.Caches().Store("fast")
	large, largeExists := ctx.Caches().Store("large")
	if !fastExists || !largeExists {
		return nil, errors.New("module cache aliases are missing")
	}
	if fast.Code() != "redis" || large.Code() != "files" {
		return nil, errors.New("module cache aliases resolved wrong stores")
	}
	binding, exists := ctx.Caches().Binding("fast")
	if !exists || binding.Namespace != "shared/fast" {
		return nil, errors.New("module cache binding is unavailable")
	}
	if _, exists := ctx.Caches().Store("not-bound"); exists {
		return nil, errors.New("module can resolve an unbound cache")
	}
	return registryRuntime{code: m.Code()}, nil
}

type cacheResolver map[cache.Code]cache.Store

func (r cacheResolver) Store(code cache.Code) (cache.Store, bool) {
	store, exists := r[code]
	return store, exists
}

type runtimeCacheStore struct {
	code cache.Code
}

func (s runtimeCacheStore) Code() cache.Code {
	return s.code
}

func (runtimeCacheStore) Ping(context.Context) error {
	return nil
}

func (runtimeCacheStore) Get(
	context.Context,
	string,
) ([]byte, error) {
	return nil, cache.ErrMiss
}

func (runtimeCacheStore) Set(
	context.Context,
	string,
	[]byte,
	cache.SetOptions,
) error {
	return nil
}

func (runtimeCacheStore) Exists(
	context.Context,
	string,
) (bool, error) {
	return false, nil
}

func (runtimeCacheStore) Delete(context.Context, string) error {
	return nil
}

func (runtimeCacheStore) InvalidateTag(
	context.Context,
	cache.Tag,
) error {
	return nil
}

func (runtimeCacheStore) Close() error {
	return nil
}

type runtimeFilesystemResolver map[filesystem.Code]filesystem.Disk

func (r runtimeFilesystemResolver) Disk(
	code filesystem.Code,
) (filesystem.Disk, bool) {
	disk, exists := r[code]
	return disk, exists
}

type runtimeDisk struct {
	filesystem.Disk
	code       filesystem.Code
	visibility filesystem.Visibility
}

func (d *runtimeDisk) Code() filesystem.Code {
	return d.code
}

func (d *runtimeDisk) Visibility() filesystem.Visibility {
	return d.visibility
}

func (*fileAwareModule) Code() kernel.ModuleCode {
	return "file-aware"
}

func (m *fileAwareModule) Build(
	_ context.Context,
	ctx kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	if ctx.Files() != m.expectedFiles {
		return nil, errors.New("module did not receive configured file service")
	}
	if ctx.Media() != m.expectedMedia {
		return nil, errors.New("module did not receive configured media service")
	}
	if ctx.Users() != m.expectedUsers {
		return nil, errors.New("module did not receive configured user service")
	}
	if ctx.Groups() != m.expectedGroups {
		return nil, errors.New("module did not receive configured group service")
	}
	if ctx.Authorization() != m.expectedAuthorization {
		return nil, errors.New(
			"module did not receive configured authorizer",
		)
	}
	return registryRuntime{code: m.Code()}, nil
}

func TestProfileRuntimeInjectsCoreServicePorts(t *testing.T) {
	files := &markerFileService{}
	media := &markerMediaService{}
	users := &markerUserService{}
	groups := &markerGroupService{}
	authorization := &markerAuthorizer{}
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		kernel.RuntimeServices{
			Files:         files,
			Media:         media,
			Users:         users,
			Groups:        groups,
			Authorization: authorization,
			EventBus:      testEventBus{},
			Logger:        testRuntimeServices().Logger,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	module := &fileAwareModule{
		expectedFiles:         files,
		expectedMedia:         media,
		expectedUsers:         users,
		expectedGroups:        groups,
		expectedAuthorization: authorization,
	}
	if _, err := factory.Make(context.Background(), kernel.Profile{
		Code: "files",
		Modules: []kernel.ProfileModule{
			{Module: module},
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestProfileRuntimeInjectsOnlyBoundCacheAliases(t *testing.T) {
	resolver := cacheResolver{
		"redis": runtimeCacheStore{code: "redis"},
		"files": runtimeCacheStore{code: "files"},
	}
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		kernel.RuntimeServices{
			Caches:   resolver,
			EventBus: testEventBus{},
			Logger:   testRuntimeServices().Logger,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := factory.Make(
		context.Background(),
		kernel.Profile{
			Code: "cached",
			Modules: []kernel.ProfileModule{{
				Module: &cacheAwareModule{},
				Caches: []cache.Binding{
					{
						Alias:     "fast",
						Code:      "redis",
						Namespace: "shared/fast",
					},
					{Alias: "large", Code: "files"},
				},
			}},
		},
	); err != nil {
		t.Fatal(err)
	}
}

func TestProfileRuntimeInjectsScopedInfrastructure(t *testing.T) {
	bus := testEventBus{}
	public := &runtimeDisk{
		code:       "public",
		visibility: filesystem.VisibilityPublic,
	}
	private := &runtimeDisk{
		code:       "private",
		visibility: filesystem.VisibilityPrivate,
	}
	module := &infrastructureAwareModule{
		eventBus: bus,
		disk:     public,
	}
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		kernel.RuntimeServices{
			Filesystems: runtimeFilesystemResolver{
				"public":  public,
				"private": private,
			},
			EventBus: bus,
			Logger:   testRuntimeServices().Logger,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	runtime, err := factory.Make(
		context.Background(),
		kernel.Profile{
			Code: "infrastructure",
			Modules: []kernel.ProfileModule{{
				Module: module,
				Filesystems: []filesystem.Binding{{
					Alias: "assets",
					Code:  "public",
				}},
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	profile := runtime.Profile()
	profile.Modules[0].Filesystems[0].Alias = "changed"
	if runtime.Profile().Modules[0].Filesystems[0].Alias != "assets" {
		t.Fatal("profile filesystem bindings share caller memory")
	}
}

func TestProfileRuntimeScopesModuleLogger(t *testing.T) {
	var output strings.Builder
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		kernel.RuntimeServices{
			EventBus: testEventBus{},
			Logger:   slog.New(slog.NewJSONHandler(&output, nil)),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := factory.Make(context.Background(), kernel.Profile{
		Code: "profile-one",
		Modules: []kernel.ProfileModule{{
			Module: loggerAwareModule{code: "module-one"},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		`"profile.code":"profile-one"`,
		`"module.code":"module-one"`,
		`"msg":"module log"`,
	} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("scoped log does not contain %q: %s", expected, output.String())
		}
	}

	if _, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		kernel.RuntimeServices{},
	); err == nil || !strings.Contains(err.Error(), "runtime logger is nil") {
		t.Fatalf("nil runtime logger error = %v", err)
	}
	if _, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		kernel.RuntimeServices{
			Logger: testRuntimeServices().Logger,
		},
	); err == nil || !strings.Contains(err.Error(), "runtime event bus is nil") {
		t.Fatalf("nil runtime event bus error = %v", err)
	}
}

func (r registryRuntime) ModuleCode() kernel.ModuleCode {
	return r.code
}

type customFieldType struct {
	code field.TypeCode
}

func (t customFieldType) Code() field.TypeCode {
	return t.code
}

func (customFieldType) Compile(any) (field.ValueType, error) {
	return customValueType{}, nil
}

type customValueType struct{}

func (customValueType) Normalize(value any) (any, error) {
	result, ok := value.(string)
	if !ok {
		return nil, errors.New("expected string")
	}
	return result, nil
}

func (customValueType) Empty(value any) bool {
	return value == ""
}

func (customValueType) Validate(any) error {
	return nil
}

func (customValueType) Rules() []string {
	return nil
}

func (customValueType) Example() any {
	return "example"
}

type customResourceType struct {
	code     resourcetype.Code
	pathMode resourcetype.PathMode
}

func (t customResourceType) Code() resourcetype.Code {
	return t.code
}

func (t customResourceType) PathMode() resourcetype.PathMode {
	return t.pathMode
}

func (customResourceType) Normalize(
	payload resourcetype.Payload,
) (resourcetype.Payload, error) {
	return payload, nil
}

func TestProfileRuntimeCollectsFieldTypesBeforeModuleBuild(t *testing.T) {
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		testRuntimeServices(),
	)
	if err != nil {
		t.Fatal(err)
	}

	profile := kernel.Profile{
		Code: "custom",
		Modules: []kernel.ProfileModule{
			{
				Module: registryModule{
					code:       "consumer",
					expectType: "custom",
				},
			},
			{
				Module: registryModule{
					code: "provider",
					fieldTypes: []field.Type{
						customFieldType{code: "custom"},
					},
				},
			},
		},
		Params: []field.Definition{
			{
				Key:   "custom_value",
				Type:  "custom",
				Label: "Custom value",
			},
		},
	}

	runtime, err := factory.Make(context.Background(), profile)
	if err != nil {
		t.Fatal(err)
	}

	if _, exists := runtime.Registry().FieldType("custom"); !exists {
		t.Fatal("custom field type is not available in runtime registry")
	}
	values, err := runtime.ParamSchema().Validate(map[string]any{
		"custom_value": "saved",
	})
	if err != nil || values["custom_value"] != "saved" {
		t.Fatalf("custom field validation = %#v, %v", values, err)
	}

	profile.Params[0].Rules = []string{"max=1"}
	if len(runtime.Profile().Params[0].Rules) != 0 {
		t.Fatal("runtime profile params share caller memory")
	}
}

func TestProfileRuntimeCollectsDeclaredPermissionsBeforeBuild(
	t *testing.T,
) {
	t.Parallel()

	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		testRuntimeServices(),
	)
	if err != nil {
		t.Fatal(err)
	}
	code := permission.MustCode(
		"provider",
		"widget",
		permission.Read,
	)
	runtime, err := factory.Make(
		context.Background(),
		kernel.Profile{
			Code: "permissions",
			Modules: []kernel.ProfileModule{
				{
					Module: registryModule{
						code:             "consumer",
						expectPermission: code,
					},
				},
				{
					Module: registryModule{
						code: "provider",
						permissionEntities: []permission.Entity{
							{Code: "widget"},
						},
					},
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := runtime.Registry().Permission(code); !exists {
		t.Fatalf("permission %q missing from runtime registry", code)
	}
}

func TestProfileRuntimeCollectsOnlyProfileModuleWidgets(t *testing.T) {
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		testRuntimeServices(),
	)
	if err != nil {
		t.Fatal(err)
	}

	current, err := factory.Make(
		context.Background(),
		kernel.Profile{
			Code: "with-widgets",
			Modules: []kernel.ProfileModule{{
				Module: widgetProviderModule{
					code: "content",
					widgets: []widget.Widget{runtimeWidget{
						definition: widget.Definition{
							Code:        "summary",
							Label:       "Summary",
							Description: "Article summary",
						},
					}},
				},
			}},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := current.Widget("content_summary"); !exists {
		t.Fatal("profile widget is unavailable")
	}
	definitions := current.Widgets()
	if len(definitions) != 1 ||
		definitions[0].Code != "content_summary" {
		t.Fatalf("profile widgets = %#v", definitions)
	}
	definitions[0].Label = "Changed"
	if current.Widgets()[0].Label != "Summary" {
		t.Fatal("profile widgets share caller memory")
	}

	empty, err := factory.Make(
		context.Background(),
		kernel.Profile{Code: "without-widgets"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := empty.Widget("content_summary"); exists ||
		len(empty.Widgets()) != 0 {
		t.Fatalf("unconnected widget leaked into profile: %#v", empty.Widgets())
	}
}

func TestProfileRuntimeRejectsInvalidFieldRegistrations(t *testing.T) {
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		testRuntimeServices(),
	)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name     string
		profile  kernel.Profile
		contains string
	}{
		{
			name: "duplicate",
			profile: kernel.Profile{
				Code: "duplicate",
				Modules: []kernel.ProfileModule{
					{
						Module: registryModule{
							code: "first",
							fieldTypes: []field.Type{
								customFieldType{code: "custom"},
							},
						},
					},
					{
						Module: registryModule{
							code: "second",
							fieldTypes: []field.Type{
								customFieldType{code: "custom"},
							},
						},
					},
				},
			},
			contains: "already exists",
		},
		{
			name: "empty code",
			profile: kernel.Profile{
				Code: "empty",
				Modules: []kernel.ProfileModule{
					{
						Module: registryModule{
							code: "provider",
							fieldTypes: []field.Type{
								customFieldType{},
							},
						},
					},
				},
			},
			contains: "code is empty",
		},
		{
			name: "unknown",
			profile: kernel.Profile{
				Code: "unknown",
				Modules: []kernel.ProfileModule{
					{Module: registryModule{code: "module"}},
				},
				Params: []field.Definition{
					{
						Key: "value", Type: "missing", Label: "Value",
					},
				},
			},
			contains: "unknown type",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := factory.Make(
				context.Background(),
				testCase.profile,
			)
			if err == nil || !strings.Contains(err.Error(), testCase.contains) {
				t.Fatalf("make error = %v", err)
			}
		})
	}
}

func TestProfileRuntimeCollectsResourceTypesBeforeModuleBuild(
	t *testing.T,
) {
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		testRuntimeServices(),
	)
	if err != nil {
		t.Fatal(err)
	}

	runtime, err := factory.Make(context.Background(), kernel.Profile{
		Code: "custom",
		Modules: []kernel.ProfileModule{
			{
				Module: registryModule{
					code:               "consumer",
					expectResourceType: "custom",
				},
			},
			{
				Module: registryModule{
					code: "provider",
					resourceTypes: []resourcetype.Type{
						customResourceType{
							code:     "custom",
							pathMode: resourcetype.PathNone,
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	resourceType, exists := runtime.Registry().ResourceType("custom")
	if !exists || resourceType.PathMode() != resourcetype.PathNone {
		t.Fatalf("custom resource type = %#v, %t", resourceType, exists)
	}
}

func TestProfileRuntimeRejectsInvalidResourceTypeRegistrations(
	t *testing.T,
) {
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		testRuntimeServices(),
	)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name     string
		types    []resourcetype.Type
		contains string
	}{
		{
			name: "empty code",
			types: []resourcetype.Type{
				customResourceType{pathMode: resourcetype.PathRoute},
			},
			contains: "code is empty",
		},
		{
			name: "duplicate",
			types: []resourcetype.Type{
				customResourceType{
					code:     "custom",
					pathMode: resourcetype.PathRoute,
				},
				customResourceType{
					code:     "custom",
					pathMode: resourcetype.PathRoute,
				},
			},
			contains: "already exists",
		},
		{
			name: "invalid path mode",
			types: []resourcetype.Type{
				customResourceType{
					code:     "custom",
					pathMode: "invalid",
				},
			},
			contains: "invalid path mode",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := factory.Make(
				context.Background(),
				kernel.Profile{
					Code: "custom",
					Modules: []kernel.ProfileModule{{
						Module: registryModule{
							code:          "provider",
							resourceTypes: testCase.types,
						},
					}},
				},
			)
			if err == nil || !strings.Contains(
				err.Error(),
				testCase.contains,
			) {
				t.Fatalf("make error = %v", err)
			}
		})
	}
}

func TestProfileRuntimeCompilesAndClonesTemplates(t *testing.T) {
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		testRuntimeServices(),
	)
	if err != nil {
		t.Fatal(err)
	}

	required := true
	profile := kernel.Profile{
		Code: "templates",
		Modules: []kernel.ProfileModule{{
			Module: registryModule{
				code: "fields",
				fieldTypes: append(
					field.StandardTypes(),
					customFieldType{code: "custom"},
				),
			},
		}},
		Templates: []template.Definition{{
			Code:  "article",
			Label: "Article",
			Fields: []field.Definition{
				{
					Key:      "headline",
					Type:     field.TypeString,
					Label:    "Headline",
					Required: &required,
					Rules:    []string{"min=2"},
				},
				{
					Key:   "custom_value",
					Type:  "custom",
					Label: "Custom value",
				},
				{
					Key:   "layout",
					Type:  field.TypeSelect,
					Label: "Layout",
					Options: field.SelectOptions{
						Choices: []field.Choice{{
							Value: "wide",
							Label: "Wide",
						}},
					},
				},
			},
		}},
	}

	runtime, err := factory.Make(context.Background(), profile)
	if err != nil {
		t.Fatal(err)
	}
	article, exists := runtime.Template("article")
	if !exists {
		t.Fatal("article template is missing")
	}
	values, err := article.FieldSchema().Validate(map[string]any{
		"headline":     "News",
		"custom_value": "custom",
		"layout":       "wide",
	})
	if err != nil ||
		values["headline"] != "News" ||
		values["custom_value"] != "custom" ||
		values["layout"] != "wide" {
		t.Fatalf("template settings = %#v, %v", values, err)
	}

	profile.Templates[0].Label = "Changed"
	profile.Templates[0].Fields[0].Rules[0] = "max=1"
	options := profile.Templates[0].Fields[2].Options.(field.SelectOptions)
	options.Choices[0].Label = "Changed"
	definition := article.Definition()
	definitionOptions := definition.Fields[2].Options.(field.SelectOptions)
	if definition.Label != "Article" ||
		definition.Fields[0].Rules[0] != "min=2" ||
		definitionOptions.Choices[0].Label != "Wide" {
		t.Fatalf("template shares caller memory: %#v", definition)
	}
	definition.Fields[0].Rules[0] = "max=1"
	if article.Definition().Fields[0].Rules[0] != "min=2" {
		t.Fatal("template definition result is mutable")
	}
}

func TestProfileRuntimeRejectsInvalidTemplates(t *testing.T) {
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		testRuntimeServices(),
	)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		name      string
		templates []template.Definition
		contains  string
	}{
		{
			name: "duplicate",
			templates: []template.Definition{
				{Code: "article", Label: "Article"},
				{Code: "article", Label: "Other"},
			},
			contains: "duplicate template code",
		},
		{
			name: "unknown field type",
			templates: []template.Definition{{
				Code:  "article",
				Label: "Article",
				Fields: []field.Definition{{
					Key: "value", Type: "missing", Label: "Value",
				}},
			}},
			contains: "unknown type",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := factory.Make(
				context.Background(),
				kernel.Profile{
					Code: "templates",
					Modules: []kernel.ProfileModule{{
						Module: registryModule{
							code:       "fields",
							fieldTypes: field.StandardTypes(),
						},
					}},
					Templates: testCase.templates,
				},
			)
			if err == nil || !strings.Contains(
				err.Error(),
				testCase.contains,
			) {
				t.Fatalf("make error = %v", err)
			}
		})
	}
}

func TestCoreModuleRegistersAllStandardFieldTypes(t *testing.T) {
	registry := core.Module{}.Registry()
	if len(registry.FieldTypes) != 10 {
		t.Fatalf("standard field types = %d", len(registry.FieldTypes))
	}

	found := make(map[field.TypeCode]bool, len(registry.FieldTypes))
	for _, fieldType := range registry.FieldTypes {
		found[fieldType.Code()] = true
	}
	for _, code := range []field.TypeCode{
		field.TypeString,
		field.TypeInteger,
		field.TypeFloat,
		field.TypeCheckbox,
		field.TypeRadio,
		field.TypeSelect,
		field.TypeTextarea,
		field.TypeEmail,
		field.TypePhone,
	} {
		if !found[code] {
			t.Fatalf("standard field type %q is missing", code)
		}
	}

	if len(registry.ResourceTypes) != 3 {
		t.Fatalf(
			"standard resource types = %d",
			len(registry.ResourceTypes),
		)
	}
	resourceTypes := make(
		map[resourcetype.Code]bool,
		len(registry.ResourceTypes),
	)
	for _, resourceType := range registry.ResourceTypes {
		resourceTypes[resourceType.Code()] = true
	}
	for _, code := range []resourcetype.Code{
		resourcetype.Page,
		resourcetype.Link,
		resourcetype.ResourceLink,
	} {
		if !resourceTypes[code] {
			t.Fatalf("standard resource type %q is missing", code)
		}
	}
}
