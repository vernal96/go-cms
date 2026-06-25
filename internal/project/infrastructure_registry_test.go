package project

import (
	"context"
	"testing"

	"github.com/vernal96/go-cms/core"
)

func TestInfrastructureRegistryStoresResourceFieldValueRepository(t *testing.T) {
	registry := NewInfrastructureRegistry()
	repository := &testResourceFieldValueRepository{}

	registry.UseResourceFieldValueRepository(repository)

	if registry.ResourceFieldValueRepository() != repository {
		t.Fatal("registry returned a different resource field value repository")
	}
}

func TestInfrastructureRegistryIgnoresNilResourceFieldValueRepository(t *testing.T) {
	registry := NewInfrastructureRegistry()
	repository := &testResourceFieldValueRepository{}
	registry.UseResourceFieldValueRepository(repository)

	registry.UseResourceFieldValueRepository(nil)

	if registry.ResourceFieldValueRepository() != repository {
		t.Fatal("nil repository must not replace the registered repository")
	}
}

func TestInfrastructureRegistryStoresWidgetInstanceRepository(t *testing.T) {
	registry := NewInfrastructureRegistry()
	repository := &testWidgetInstanceRepository{}

	registry.UseWidgetInstanceRepository(repository)

	if registry.WidgetInstanceRepository() != repository {
		t.Fatal("registry returned a different widget instance repository")
	}
}

func TestInfrastructureRegistryIgnoresNilWidgetInstanceRepository(t *testing.T) {
	registry := NewInfrastructureRegistry()
	repository := &testWidgetInstanceRepository{}
	registry.UseWidgetInstanceRepository(repository)

	registry.UseWidgetInstanceRepository(nil)

	if registry.WidgetInstanceRepository() != repository {
		t.Fatal("nil repository must not replace the registered repository")
	}
}

type testResourceFieldValueRepository struct{}

func (*testResourceFieldValueRepository) FindByResourceID(
	ctx context.Context,
	resourceID core.ResourceID,
) ([]core.ResourceFieldValue, error) {
	return nil, nil
}

func (*testResourceFieldValueRepository) FindByResourceAndField(
	ctx context.Context,
	resourceID core.ResourceID,
	field core.ResourceFieldCode,
) (core.ResourceFieldValue, error) {
	return core.ResourceFieldValue{}, nil
}

func (*testResourceFieldValueRepository) Save(
	ctx context.Context,
	value core.ResourceFieldValue,
) (core.ResourceFieldValue, error) {
	return value, nil
}

type testWidgetInstanceRepository struct{}

func (*testWidgetInstanceRepository) FindForResource(
	ctx context.Context,
	resource core.Resource,
) ([]core.WidgetInstance, error) {
	return nil, nil
}
