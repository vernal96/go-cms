package coremodule

import (
	"testing"

	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/core/modules/core/resources"
)

func TestModuleRegistersPageResourceType(t *testing.T) {
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
}
