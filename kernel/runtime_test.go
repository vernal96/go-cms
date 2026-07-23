package kernel_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
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
	code       kernel.ModuleCode
	fieldTypes []field.Type
	expectType field.TypeCode
}

func (m registryModule) Code() kernel.ModuleCode {
	return m.code
}

func (m registryModule) Registry() kernel.ModuleRegistry {
	return kernel.ModuleRegistry{
		FieldTypes: append([]field.Type(nil), m.fieldTypes...),
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

	return registryRuntime{code: m.code}, nil
}

type registryRuntime struct {
	code kernel.ModuleCode
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
}
