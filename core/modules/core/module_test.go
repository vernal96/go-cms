package coremodule

import (
	"testing"

	"github.com/vernal96/go-cms/core"
	modulefields "github.com/vernal96/go-cms/core/modules/core/fields"
	"github.com/vernal96/go-cms/core/modules/core/resources"
)

func TestModuleRegistersPageResourceTypeAndDefaultTemplate(t *testing.T) {
	registry := core.NewRuntimeRegistry()

	if err := New(Config{}).Register(registry.ForModule(ModuleCode)); err != nil {
		t.Fatal(err)
	}

	resourceType, exists := registry.ResourceTypes().Get(resources.PageResourceTypeCode)
	if !exists {
		t.Fatal("page resource type is not registered")
	}
	if resourceType.Name() != "Page" {
		t.Fatalf("unexpected page resource type name: %q", resourceType.Name())
	}

	template, exists := registry.ResourceTemplates().Get(
		resources.PageResourceTypeCode,
		resources.PageDefaultTemplateCode,
	)
	if !exists {
		t.Fatal("default page resource template is not registered")
	}
	if template.Name() != "Default page" {
		t.Fatalf("unexpected default page resource template name: %q", template.Name())
	}

	field, exists := registry.ResourceFields().Get(
		resources.PageResourceTypeCode,
		resources.PageDefaultTemplateCode,
		resources.PageContentFieldCode,
	)
	if !exists {
		t.Fatal("content page resource field is not registered")
	}
	if field.Name() != "Content" {
		t.Fatalf("unexpected content resource field name: %q", field.Name())
	}
	if field.Field().Code() != modulefields.TextFieldTypeCode {
		t.Fatalf("unexpected content field type: %q", field.Field().Code())
	}
	if field.Required() {
		t.Fatal("content resource field must not be required")
	}
}
