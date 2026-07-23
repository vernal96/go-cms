package kernel_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	corefile "github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
)

type emptyDatabaseResolver struct{}

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
	expectType         field.TypeCode
	expectResourceType resourcetype.Code
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

	return registryRuntime{code: m.code}, nil
}

type registryRuntime struct {
	code kernel.ModuleCode
}

type markerFileService struct {
	corefile.Service
}

type fileAwareModule struct {
	expected corefile.Service
}

func (*fileAwareModule) Code() kernel.ModuleCode {
	return "file-aware"
}

func (m *fileAwareModule) Build(
	_ context.Context,
	ctx kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	if ctx.Files() != m.expected {
		return nil, errors.New("module did not receive configured file service")
	}
	return registryRuntime{code: m.Code()}, nil
}

func TestProfileRuntimeInjectsOnlyFileServicePort(t *testing.T) {
	files := &markerFileService{}
	factory, err := kernel.NewProfileRuntimeFactory(
		emptyDatabaseResolver{},
		files,
	)
	if err != nil {
		t.Fatal(err)
	}
	module := &fileAwareModule{expected: files}
	if _, err := factory.Make(context.Background(), kernel.Profile{
		Code: "files",
		Modules: []kernel.ProfileModule{
			{Module: module},
		},
	}); err != nil {
		t.Fatal(err)
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
	factory, err := kernel.NewProfileRuntimeFactory(emptyDatabaseResolver{})
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

func TestProfileRuntimeRejectsInvalidFieldRegistrations(t *testing.T) {
	factory, err := kernel.NewProfileRuntimeFactory(emptyDatabaseResolver{})
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
	factory, err := kernel.NewProfileRuntimeFactory(emptyDatabaseResolver{})
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
	factory, err := kernel.NewProfileRuntimeFactory(emptyDatabaseResolver{})
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
	factory, err := kernel.NewProfileRuntimeFactory(emptyDatabaseResolver{})
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
	factory, err := kernel.NewProfileRuntimeFactory(emptyDatabaseResolver{})
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
	if len(registry.FieldTypes) != 9 {
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
