package kernel_test

import (
	"testing"

	"github.com/vernal96/go-cms/kernel"
)

const testProfileCode kernel.ProfileCode = "test"

type testProfile struct {
	code kernel.ProfileCode
}

func (p *testProfile) Code() kernel.ProfileCode {
	return p.code
}

func (p *testProfile) Modules() []kernel.ProfileModule {
	return nil
}

func TestProfileRegistryManager_RegisterAndGet(t *testing.T) {
	registry := kernel.NewProfileRegistryManager()
	profile := &testProfile{
		code: testProfileCode,
	}

	err := registry.Register(profile)
	if err != nil {
		t.Fatalf("register profile: %v", err)
	}

	foundProfile, exists := registry.Get(testProfileCode)
	if !exists {
		t.Fatal("registered profile was not found")
	}

	if foundProfile != profile {
		t.Fatal("registry returned a different profile")
	}
}

func TestProfileRegistryManager_GetReturnsFalseForUnknownProfile(t *testing.T) {
	registry := kernel.NewProfileRegistryManager()

	_, exists := registry.Get("unknown")
	if exists {
		t.Fatal("unexpected profile found")
	}
}

func TestProfileRegistryManager_RejectsDuplicate(t *testing.T) {
	registry := kernel.NewProfileRegistryManager()

	firstProfile := &testProfile{
		code: testProfileCode,
	}

	secondProfile := &testProfile{
		code: testProfileCode,
	}

	err := registry.Register(firstProfile)
	if err != nil {
		t.Fatalf("register first profile: %v", err)
	}

	err = registry.Register(secondProfile)
	if err == nil {
		t.Fatal("expected duplicate profile error")
	}
}

func TestProfileRegistryManager_RejectsNilProfile(t *testing.T) {
	registry := kernel.NewProfileRegistryManager()

	err := registry.Register(nil)
	if err == nil {
		t.Fatal("expected nil profile error")
	}
}

func TestProfileRegistryManager_RejectsEmptyProfileCode(t *testing.T) {
	registry := kernel.NewProfileRegistryManager()

	profile := &testProfile{}

	err := registry.Register(profile)
	if err == nil {
		t.Fatal("expected empty profile code error")
	}
}
