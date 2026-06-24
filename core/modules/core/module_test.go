package coremodule

import (
	"testing"

	"github.com/vernal96/go-cms/core"
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
}
