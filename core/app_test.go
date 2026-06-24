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
	)
	if err != nil {
		t.Fatal(err)
	}

	if app.ResourceFieldValues() != resourceFieldValues {
		t.Fatal("app returned a different resource field value repository")
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
