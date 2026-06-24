package core

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestNewResourceFieldValueWriter(t *testing.T) {
	if NewResourceFieldValueWriter() == nil {
		t.Fatal("resource field value writer is nil")
	}
}

func TestResourceFieldValueWriterValidatesInput(t *testing.T) {
	writer := NewResourceFieldValueWriter()

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := writer.Save(ctx, nil, Resource{}, "", nil)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("nil runtime", func(t *testing.T) {
		_, err := writer.Save(context.Background(), nil, Resource{}, "content", nil)
		if err == nil || err.Error() != "site runtime is nil" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("resource id", func(t *testing.T) {
		runtime := newResourceFieldValueWriterRuntime(t, NewRuntimeRegistry(), nil)

		_, err := writer.Save(context.Background(), runtime, Resource{}, "content", nil)
		if err == nil || err.Error() != "resource id must be positive" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("field", func(t *testing.T) {
		runtime := newResourceFieldValueWriterRuntime(t, NewRuntimeRegistry(), nil)

		_, err := writer.Save(context.Background(), runtime, Resource{ID: 1}, "", nil)
		if err == nil || err.Error() != "resource field is empty" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestResourceFieldValueWriterValidatesRegistry(t *testing.T) {
	writer := NewResourceFieldValueWriter()
	resource := Resource{
		ID:       21,
		Type:     "page",
		Template: "default",
	}

	t.Run("resource type", func(t *testing.T) {
		runtime := newResourceFieldValueWriterRuntime(t, NewRuntimeRegistry(), nil)

		_, err := writer.Save(context.Background(), runtime, resource, "content", nil)
		if err == nil || err.Error() != `resource type "page" is not registered` {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("resource template", func(t *testing.T) {
		registry := NewRuntimeRegistry()
		if err := registry.ResourceTypes().Register(readerResourceType{}); err != nil {
			t.Fatal(err)
		}
		runtime := newResourceFieldValueWriterRuntime(t, registry, nil)

		_, err := writer.Save(context.Background(), runtime, resource, "content", nil)
		if err == nil ||
			err.Error() != `resource template "default" for resource type "page" is not registered` {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("resource field", func(t *testing.T) {
		registry := NewRuntimeRegistry()
		if err := registry.ResourceTypes().Register(readerResourceType{}); err != nil {
			t.Fatal(err)
		}
		if err := registry.ResourceTemplates().Register(readerResourceTemplate{}); err != nil {
			t.Fatal(err)
		}
		runtime := newResourceFieldValueWriterRuntime(t, registry, nil)

		_, err := writer.Save(context.Background(), runtime, resource, "content", nil)
		if err == nil ||
			err.Error() != `resource field "content" for resource type "page" and template "default" is not registered` {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestResourceFieldValueWriterSave(t *testing.T) {
	registry := NewRuntimeRegistry()
	if err := registry.ResourceTypes().Register(readerResourceType{}); err != nil {
		t.Fatal(err)
	}
	if err := registry.ResourceTemplates().Register(readerResourceTemplate{}); err != nil {
		t.Fatal(err)
	}
	if err := registry.ResourceFields().Register(readerResourceField{}); err != nil {
		t.Fatal(err)
	}

	repository := &readerResourceFieldValueRepository{}
	runtime := newResourceFieldValueWriterRuntime(t, registry, repository)
	resource := Resource{
		ID:       21,
		Type:     "page",
		Template: "default",
	}
	fieldValue := map[string]any{
		"text": "Hello",
	}

	saved, err := NewResourceFieldValueWriter().Save(
		context.Background(),
		runtime,
		resource,
		"content",
		fieldValue,
	)
	if err != nil {
		t.Fatal(err)
	}

	expected := ResourceFieldValue{
		ResourceID: resource.ID,
		Field:      "content",
		Value:      fieldValue,
	}
	if !reflect.DeepEqual(saved, expected) {
		t.Fatalf("unexpected saved value: %#v", saved)
	}
	if !reflect.DeepEqual(repository.saved, expected) {
		t.Fatalf("unexpected repository value: %#v", repository.saved)
	}
}

func newResourceFieldValueWriterRuntime(
	t *testing.T,
	registry Registry,
	fieldValues ResourceFieldValueRepository,
) *SiteRuntime {
	t.Helper()

	if fieldValues == nil {
		fieldValues = &readerResourceFieldValueRepository{}
	}

	return newResourceReaderRuntimeWithRegistry(
		t,
		&readerResourceRepository{},
		fieldValues,
		registry,
	)
}
