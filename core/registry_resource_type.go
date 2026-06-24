package core

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type runtimeResourceTypeRegistry struct {
	resourceTypes map[ResourceType]ResourceTypeDefinition
}

func (r *runtimeResourceTypeRegistry) Register(resourceType ResourceTypeDefinition) error {
	if resourceType == nil {
		return errors.New("resource type is nil")
	}

	code := resourceType.Code()
	if code == "" {
		return errors.New("resource type code is empty")
	}

	if _, exists := r.resourceTypes[code]; exists {
		return fmt.Errorf("resource type %q is already registered", code)
	}

	r.resourceTypes[code] = resourceType

	return nil
}

func (r *runtimeResourceTypeRegistry) Get(code ResourceType) (ResourceTypeDefinition, bool) {
	resourceType, exists := r.resourceTypes[code]

	return resourceType, exists
}

func (r *runtimeResourceTypeRegistry) All() []ResourceTypeDefinition {
	resourceTypes := make([]ResourceTypeDefinition, 0, len(r.resourceTypes))

	for _, resourceType := range r.resourceTypes {
		resourceTypes = append(resourceTypes, resourceType)
	}

	slices.SortFunc(resourceTypes, func(a, b ResourceTypeDefinition) int {
		return strings.Compare(string(a.Code()), string(b.Code()))
	})

	return resourceTypes
}

var _ ResourceTypeRegistry = (*runtimeResourceTypeRegistry)(nil)
