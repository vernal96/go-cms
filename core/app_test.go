package core

import (
	"context"
	"testing"
)

func TestNewAppRequiresResourceFieldValueRepository(t *testing.T) {
	app, err := NewApp(
		testCacheManager{},
		testFileStorageManager{},
		NullEventBus{},
		NullLogger{},
		testResourceRepository{},
		nil,
		testWidgetInstanceRepository{},
	)
	if err == nil {
		t.Fatal("expected nil resource field value repository error")
	}
	if app != nil {
		t.Fatal("app must be nil when resource field value repository is nil")
	}
}

func TestAppProvidesResourceFieldValueRepository(t *testing.T) {
	resourceFieldValues := &testResourceFieldValueRepository{}

	app, err := NewApp(
		testCacheManager{},
		testFileStorageManager{},
		NullEventBus{},
		NullLogger{},
		testResourceRepository{},
		resourceFieldValues,
		testWidgetInstanceRepository{},
	)
	if err != nil {
		t.Fatal(err)
	}

	if app.ResourceFieldValues() != resourceFieldValues {
		t.Fatal("app returned a different resource field value repository")
	}
}

func TestNewAppRequiresWidgetInstanceRepository(t *testing.T) {
	app, err := NewApp(
		testCacheManager{},
		testFileStorageManager{},
		NullEventBus{},
		NullLogger{},
		testResourceRepository{},
		&testResourceFieldValueRepository{},
		nil,
	)
	if err == nil {
		t.Fatal("expected nil widget instance repository error")
	}
	if app != nil {
		t.Fatal("app must be nil when widget instance repository is nil")
	}
}

func TestAppProvidesWidgetInstanceRepository(t *testing.T) {
	widgetInstances := &testWidgetInstanceRepository{}

	app, err := NewApp(
		testCacheManager{},
		testFileStorageManager{},
		NullEventBus{},
		NullLogger{},
		testResourceRepository{},
		&testResourceFieldValueRepository{},
		widgetInstances,
	)
	if err != nil {
		t.Fatal(err)
	}

	if app.WidgetInstances() != widgetInstances {
		t.Fatal("app returned a different widget instance repository")
	}
}

type testCacheManager struct{}

func (testCacheManager) Store(name CacheStoreName) (CacheStore, error) {
	return NullCacheStore{}, nil
}

func (testCacheManager) Scope(scope CacheScope) (CacheStore, error) {
	return NullCacheStore{}, nil
}

type testFileStorageManager struct{}

func (testFileStorageManager) Disk(name FileDisk) (FileStorage, error) {
	return NullFileStorage{}, nil
}

type testResourceRepository struct{}

func (testResourceRepository) FindByID(
	ctx context.Context,
	id ResourceID,
) (Resource, error) {
	return Resource{}, nil
}

func (testResourceRepository) FindByPath(
	ctx context.Context,
	siteID int64,
	path string,
) (Resource, error) {
	return Resource{}, nil
}

func (testResourceRepository) FindChildren(
	ctx context.Context,
	parentID ResourceID,
) ([]Resource, error) {
	return nil, nil
}

type testResourceFieldValueRepository struct{}

func (*testResourceFieldValueRepository) FindByResourceID(
	ctx context.Context,
	resourceID ResourceID,
) ([]ResourceFieldValue, error) {
	return nil, nil
}

func (*testResourceFieldValueRepository) FindByResourceAndField(
	ctx context.Context,
	resourceID ResourceID,
	field ResourceFieldCode,
) (ResourceFieldValue, error) {
	return ResourceFieldValue{}, nil
}

func (*testResourceFieldValueRepository) Save(
	ctx context.Context,
	value ResourceFieldValue,
) (ResourceFieldValue, error) {
	return value, nil
}

type testWidgetInstanceRepository struct{}

func (testWidgetInstanceRepository) FindForResource(
	ctx context.Context,
	resource Resource,
) ([]WidgetInstance, error) {
	return nil, nil
}
