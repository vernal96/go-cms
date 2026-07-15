package kernel_test

import (
	"testing"

	"github.com/vernal96/go-cms/kernel"
)

const (
	testRepositoryContract kernel.AdapterContractCode = "test.repository"
	testMemoryAdapter      kernel.AdapterCode         = "memory"
)

type testRepository interface {
	Name() string
}

type memoryRepository struct{}

func (r *memoryRepository) Name() string {
	return "memory"
}

func TestAdapterRegistry_AddAndGet(t *testing.T) {
	registry := kernel.NewAdapterRegistry()
	repository := &memoryRepository{}

	err := registry.Add(
		testRepositoryContract,
		testMemoryAdapter,
		repository,
	)
	if err != nil {
		t.Fatalf("add adapter: %v", err)
	}

	foundRepository, err := kernel.AdapterAs[testRepository](
		registry,
		testRepositoryContract,
		testMemoryAdapter,
	)
	if err != nil {
		t.Fatalf("get adapter: %v", err)
	}

	if foundRepository != repository {
		t.Fatal("registry returned a different adapter")
	}
}

func TestAdapterRegistry_RejectsDuplicate(t *testing.T) {
	registry := kernel.NewAdapterRegistry()
	repository := &memoryRepository{}

	err := registry.Add(
		testRepositoryContract,
		testMemoryAdapter,
		repository,
	)
	if err != nil {
		t.Fatalf("add first adapter: %v", err)
	}

	err = registry.Add(
		testRepositoryContract,
		testMemoryAdapter,
		repository,
	)
	if err == nil {
		t.Fatal("expected duplicate adapter error")
	}
}

func TestAdapterAs_ReturnsErrorWhenAdapterIsNotRegistered(t *testing.T) {
	registry := kernel.NewAdapterRegistry()

	_, err := kernel.AdapterAs[testRepository](
		registry,
		testRepositoryContract,
		testMemoryAdapter,
	)
	if err == nil {
		t.Fatal("expected adapter not registered error")
	}
}

func TestAdapterAs_ReturnsErrorForInvalidType(t *testing.T) {
	registry := kernel.NewAdapterRegistry()

	err := registry.Add(
		testRepositoryContract,
		testMemoryAdapter,
		"not a repository",
	)
	if err != nil {
		t.Fatalf("add adapter: %v", err)
	}

	_, err = kernel.AdapterAs[testRepository](
		registry,
		testRepositoryContract,
		testMemoryAdapter,
	)
	if err == nil {
		t.Fatal("expected invalid adapter type error")
	}
}
