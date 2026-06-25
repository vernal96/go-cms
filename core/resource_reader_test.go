package core

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/vernal96/go-cms/core/fields"
)

func TestNewResourceReader(t *testing.T) {
	if NewResourceReader() == nil {
		t.Fatal("resource reader is nil")
	}
}

func TestResourceReaderValidatesInput(t *testing.T) {
	reader := NewResourceReader()

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := reader.ReadByPath(ctx, nil, "/")
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("nil runtime", func(t *testing.T) {
		_, err := reader.ReadByPath(context.Background(), nil, "/")
		if err == nil || err.Error() != "site runtime is nil" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("empty path", func(t *testing.T) {
		runtime := newResourceReaderRuntime(t)

		_, err := reader.ReadByPath(context.Background(), runtime, "")
		if err == nil || err.Error() != "resource path is empty" {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("path without slash", func(t *testing.T) {
		runtime := newResourceReaderRuntime(t)

		_, err := reader.ReadByPath(context.Background(), runtime, "home")
		if err == nil || err.Error() != "resource path must start with /" {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestResourceReaderReadByPath(t *testing.T) {
	resource := Resource{
		ID:       21,
		SiteID:   7,
		Type:     "page",
		Template: "default",
		Title:    "Home",
		Path:     "/",
	}
	values := []ResourceFieldValue{
		{
			ResourceID: resource.ID,
			Field:      "content",
			Value:      map[string]any{"text": "Hello"},
		},
	}
	resources := &readerResourceRepository{
		resource: resource,
	}
	fieldValues := &readerResourceFieldValueRepository{
		values: values,
	}
	widgets := []WidgetInstance{
		{
			ID:         30,
			Source:     WidgetInstanceSourceResource,
			ResourceID: resource.ID,
			Widget:     "core.text",
			Area:       "main",
			Params: WidgetParams{
				"text": "Widget",
			},
		},
	}
	widgetInstances := &readerWidgetInstanceRepository{
		instances: widgets,
	}
	runtime := newResourceReaderRuntimeWithDependencies(
		t,
		resources,
		fieldValues,
		widgetInstances,
		newResourceReaderRegistry(t),
	)

	data, err := NewResourceReader().ReadByPath(context.Background(), runtime, "/")
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(data.Resource, resource) {
		t.Fatalf("unexpected resource: %#v", data.Resource)
	}
	if len(data.Fields) != 1 || data.Fields[0].Code() != "content" {
		t.Fatalf("unexpected fields: %#v", data.Fields)
	}
	if !reflect.DeepEqual(data.Values, values) {
		t.Fatalf("unexpected values: %#v", data.Values)
	}
	if !reflect.DeepEqual(data.Widgets, widgets) {
		t.Fatalf("unexpected widgets: %#v", data.Widgets)
	}
	if resources.siteID != 7 || resources.path != "/" {
		t.Fatalf(
			"unexpected resource query: site id %d, path %q",
			resources.siteID,
			resources.path,
		)
	}
	if fieldValues.resourceID != resource.ID {
		t.Fatalf("unexpected field values resource id: %d", fieldValues.resourceID)
	}
	if !reflect.DeepEqual(widgetInstances.resource, resource) {
		t.Fatalf("unexpected widget instance resource: %#v", widgetInstances.resource)
	}
}

func TestResourceReaderRejectsUnregisteredResourceType(t *testing.T) {
	resources := &readerResourceRepository{
		resource: Resource{
			ID:       21,
			SiteID:   7,
			Type:     "page",
			Template: "default",
			Path:     "/",
		},
	}
	fieldValues := &readerResourceFieldValueRepository{}
	runtime := newResourceReaderRuntimeWithRegistry(
		t,
		resources,
		fieldValues,
		NewRuntimeRegistry(),
	)

	_, err := NewResourceReader().ReadByPath(context.Background(), runtime, "/")
	if err == nil || err.Error() != `resource type "page" is not registered` {
		t.Fatalf("unexpected error: %v", err)
	}
	if fieldValues.resourceID != 0 {
		t.Fatal("field values must not be read for an unregistered resource type")
	}
}

func TestResourceReaderRejectsUnregisteredResourceTemplate(t *testing.T) {
	resources := &readerResourceRepository{
		resource: Resource{
			ID:       21,
			SiteID:   7,
			Type:     "page",
			Template: "default",
			Path:     "/",
		},
	}
	fieldValues := &readerResourceFieldValueRepository{}
	registry := NewRuntimeRegistry()
	if err := registry.ResourceTypes().Register(readerResourceType{}); err != nil {
		t.Fatal(err)
	}
	runtime := newResourceReaderRuntimeWithRegistry(
		t,
		resources,
		fieldValues,
		registry,
	)

	_, err := NewResourceReader().ReadByPath(context.Background(), runtime, "/")
	if err == nil ||
		err.Error() != `resource template "default" for resource type "page" is not registered` {
		t.Fatalf("unexpected error: %v", err)
	}
	if fieldValues.resourceID != 0 {
		t.Fatal("field values must not be read for an unregistered resource template")
	}
}

func newResourceReaderRuntime(t *testing.T) *SiteRuntime {
	t.Helper()

	return newResourceReaderRuntimeWithRepositories(
		t,
		&readerResourceRepository{},
		&readerResourceFieldValueRepository{},
	)
}

func newResourceReaderRuntimeWithRepositories(
	t *testing.T,
	resources ResourceRepository,
	fieldValues ResourceFieldValueRepository,
) *SiteRuntime {
	t.Helper()

	return newResourceReaderRuntimeWithDependencies(
		t,
		resources,
		fieldValues,
		&readerWidgetInstanceRepository{},
		newResourceReaderRegistry(t),
	)
}

func newResourceReaderRegistry(t *testing.T) *RuntimeRegistry {
	t.Helper()

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

	return registry
}

func newResourceReaderRuntimeWithRegistry(
	t *testing.T,
	resources ResourceRepository,
	fieldValues ResourceFieldValueRepository,
	registry Registry,
) *SiteRuntime {
	t.Helper()

	return newResourceReaderRuntimeWithDependencies(
		t,
		resources,
		fieldValues,
		&readerWidgetInstanceRepository{},
		registry,
	)
}

func newResourceReaderRuntimeWithDependencies(
	t *testing.T,
	resources ResourceRepository,
	fieldValues ResourceFieldValueRepository,
	widgetInstances WidgetInstanceRepository,
	registry Registry,
) *SiteRuntime {
	t.Helper()

	app, err := NewApp(
		readerCacheManager{},
		readerFileStorageManager{},
		NullEventBus{},
		NullLogger{},
		resources,
		fieldValues,
		widgetInstances,
	)
	if err != nil {
		t.Fatal(err)
	}

	runtime, err := NewSiteRuntime(
		app,
		Site{
			ID:          7,
			ProfileCode: "main",
		},
		readerSiteProfile{},
		registry,
	)
	if err != nil {
		t.Fatal(err)
	}

	return runtime
}

type readerCacheManager struct{}

func (readerCacheManager) Store(name CacheStoreName) (CacheStore, error) {
	return NullCacheStore{}, nil
}

func (readerCacheManager) Scope(scope CacheScope) (CacheStore, error) {
	return NullCacheStore{}, nil
}

type readerFileStorageManager struct{}

func (readerFileStorageManager) Disk(name FileDisk) (FileStorage, error) {
	return NullFileStorage{}, nil
}

type readerResourceRepository struct {
	resource Resource
	siteID   int64
	path     string
}

func (r *readerResourceRepository) FindByID(
	ctx context.Context,
	id ResourceID,
) (Resource, error) {
	return Resource{}, nil
}

func (r *readerResourceRepository) FindByPath(
	ctx context.Context,
	siteID int64,
	path string,
) (Resource, error) {
	r.siteID = siteID
	r.path = path

	return r.resource, nil
}

func (r *readerResourceRepository) FindChildren(
	ctx context.Context,
	parentID ResourceID,
) ([]Resource, error) {
	return nil, nil
}

type readerResourceFieldValueRepository struct {
	values     []ResourceFieldValue
	resourceID ResourceID
	saved      ResourceFieldValue
	saveErr    error
}

func (r *readerResourceFieldValueRepository) FindByResourceID(
	ctx context.Context,
	resourceID ResourceID,
) ([]ResourceFieldValue, error) {
	r.resourceID = resourceID

	return r.values, nil
}

func (r *readerResourceFieldValueRepository) FindByResourceAndField(
	ctx context.Context,
	resourceID ResourceID,
	field ResourceFieldCode,
) (ResourceFieldValue, error) {
	return ResourceFieldValue{}, nil
}

func (r *readerResourceFieldValueRepository) Save(
	ctx context.Context,
	value ResourceFieldValue,
) (ResourceFieldValue, error) {
	r.saved = value
	if r.saveErr != nil {
		return ResourceFieldValue{}, r.saveErr
	}

	return value, nil
}

type readerWidgetInstanceRepository struct {
	instances []WidgetInstance
	resource  Resource
	err       error
}

func (r *readerWidgetInstanceRepository) FindForResource(
	ctx context.Context,
	resource Resource,
) ([]WidgetInstance, error) {
	r.resource = resource
	if r.err != nil {
		return nil, r.err
	}

	return r.instances, nil
}

type readerSiteProfile struct{}

func (readerSiteProfile) Code() string {
	return "main"
}

func (readerSiteProfile) Modules() []Module {
	return nil
}

type readerResourceType struct{}

func (readerResourceType) Code() ResourceType {
	return "page"
}

func (readerResourceType) Name() string {
	return "Page"
}

type readerResourceTemplate struct{}

func (readerResourceTemplate) Code() ResourceTemplateCode {
	return "default"
}

func (readerResourceTemplate) Name() string {
	return "Default page"
}

func (readerResourceTemplate) ResourceType() ResourceType {
	return "page"
}

type readerResourceField struct{}

func (readerResourceField) Code() ResourceFieldCode {
	return "content"
}

func (readerResourceField) Name() string {
	return "Content"
}

func (readerResourceField) Field() fields.FieldType {
	return fields.NewInput()
}

func (readerResourceField) ResourceType() ResourceType {
	return "page"
}

func (readerResourceField) ResourceTemplate() ResourceTemplateCode {
	return "default"
}

func (readerResourceField) Required() bool {
	return false
}
