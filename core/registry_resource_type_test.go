package core

import (
	"strings"
	"testing"
)

type testResourceType struct {
	code ResourceType
	name string
}

func (r testResourceType) Code() ResourceType {
	return r.code
}

func (r testResourceType) Name() string {
	return r.name
}

func TestRuntimeResourceTypeRegistryRegister(t *testing.T) {
	registry := NewRuntimeRegistry().ResourceTypes()
	resourceType := testResourceType{
		code: "page",
		name: "Page",
	}

	if err := registry.Register(resourceType); err != nil {
		t.Fatal(err)
	}

	registered, exists := registry.Get("page")
	if !exists {
		t.Fatal("registered resource type not found")
	}
	if registered.Name() != "Page" {
		t.Fatalf("unexpected resource type name: %q", registered.Name())
	}

	all := registry.All()
	if len(all) != 1 || all[0].Code() != "page" {
		t.Fatalf("unexpected registered resource types: %#v", all)
	}
}

func TestRuntimeResourceTypeRegistryRejectsDuplicate(t *testing.T) {
	registry := NewRuntimeRegistry().ResourceTypes()
	resourceType := testResourceType{
		code: "page",
		name: "Page",
	}

	if err := registry.Register(resourceType); err != nil {
		t.Fatal(err)
	}

	err := registry.Register(resourceType)
	if err == nil {
		t.Fatal("expected duplicate resource type error")
	}
	if !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuntimeResourceTypeRegistryValidatesResourceType(t *testing.T) {
	t.Run("nil resource type", func(t *testing.T) {
		err := NewRuntimeRegistry().ResourceTypes().Register(nil)
		if err == nil {
			t.Fatal("expected nil resource type error")
		}
	})

	t.Run("empty code", func(t *testing.T) {
		err := NewRuntimeRegistry().ResourceTypes().Register(testResourceType{
			name: "Missing code",
		})
		if err == nil {
			t.Fatal("expected empty resource type code error")
		}
	})
}
