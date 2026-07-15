package kernel_test

import (
	"testing"

	"github.com/vernal96/go-cms/kernel"
)

type moduleTestConfig struct {
	Name string
}

func TestModuleConfigFrom_ReturnsValueConfig(t *testing.T) {
	wantConfig := moduleTestConfig{
		Name: "value",
	}

	moduleContext := kernel.NewModuleContext(nil, nil, wantConfig)
	gotConfig, err := kernel.ModuleConfigFrom[moduleTestConfig](moduleContext)
	if err != nil {
		t.Fatalf("get module config: %v", err)
	}

	if gotConfig != wantConfig {
		t.Fatalf("unexpected module config: got %#v, want %#v", gotConfig, wantConfig)
	}
}

func TestModuleConfigFrom_DereferencesPointerConfig(t *testing.T) {
	wantConfig := moduleTestConfig{
		Name: "pointer",
	}

	moduleContext := kernel.NewModuleContext(nil, nil, &wantConfig)
	gotConfig, err := kernel.ModuleConfigFrom[moduleTestConfig](moduleContext)
	if err != nil {
		t.Fatalf("get module config: %v", err)
	}

	if gotConfig != wantConfig {
		t.Fatalf("unexpected module config: got %#v, want %#v", gotConfig, wantConfig)
	}
}

func TestModuleConfigFrom_ReturnsZeroValueForNilConfig(t *testing.T) {
	moduleContext := kernel.NewModuleContext(nil, nil, nil)
	gotConfig, err := kernel.ModuleConfigFrom[moduleTestConfig](moduleContext)
	if err != nil {
		t.Fatalf("get module config: %v", err)
	}

	if gotConfig != (moduleTestConfig{}) {
		t.Fatalf("expected zero config, got %#v", gotConfig)
	}
}

func TestModuleConfigFrom_ReturnsErrorForInvalidType(t *testing.T) {
	moduleContext := kernel.NewModuleContext(nil, nil, "invalid config")

	_, err := kernel.ModuleConfigFrom[moduleTestConfig](moduleContext)
	if err == nil {
		t.Fatal("expected invalid module config type error")
	}
}

func TestModuleConfigFrom_ReturnsErrorForNilPointer(t *testing.T) {
	var config *moduleTestConfig
	moduleContext := kernel.NewModuleContext(nil, nil, config)

	_, err := kernel.ModuleConfigFrom[moduleTestConfig](moduleContext)
	if err == nil {
		t.Fatal("expected nil module config error")
	}
}

