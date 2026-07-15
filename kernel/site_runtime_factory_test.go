package kernel_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/vernal96/go-cms/kernel"
)

type runtimeTestProfile struct {
	code    kernel.ProfileCode
	modules []kernel.ProfileModule
}

func (p *runtimeTestProfile) Code() kernel.ProfileCode {
	return p.code
}

func (p *runtimeTestProfile) Modules() []kernel.ProfileModule {
	return p.modules
}

type recordingModule struct {
	code             kernel.ModuleCode
	events           *[]string
	registerErr      error
	bootErr          error
	registerRegistry kernel.Registry
	bootContext      kernel.ModuleContext
}

func (m *recordingModule) Code() kernel.ModuleCode {
	return m.code
}

func (m *recordingModule) Register(registry kernel.Registry) error {
	*m.events = append(*m.events, "register:"+string(m.code))
	m.registerRegistry = registry

	return m.registerErr
}

func (m *recordingModule) Boot(ctx context.Context, moduleContext kernel.ModuleContext) error {
	_ = ctx

	*m.events = append(*m.events, "boot:"+string(m.code))
	m.bootContext = moduleContext

	return m.bootErr
}

func TestSiteRuntimeFactory_MakeRegistersAllModulesBeforeBoot(t *testing.T) {
	app := kernel.NewApp(kernel.AppConfig{})
	events := make([]string, 0)

	firstModule := &recordingModule{
		code:   "first",
		events: &events,
	}

	secondModule := &recordingModule{
		code:   "second",
		events: &events,
	}

	firstConfig := struct {
		Value string
	}{
		Value: "first config",
	}

	secondConfig := struct {
		Value string
	}{
		Value: "second config",
	}

	profile := &runtimeTestProfile{
		code: "test",
		modules: []kernel.ProfileModule{
			{
				Module:       firstModule,
				ModuleConfig: firstConfig,
			},
			{
				Module:       secondModule,
				ModuleConfig: secondConfig,
			},
		},
	}

	factory := kernel.NewSiteRuntimeFactory(app)
	runtime, err := factory.Make(context.Background(), profile)
	if err != nil {
		t.Fatalf("make site runtime: %v", err)
	}

	wantEvents := []string{
		"register:first",
		"register:second",
		"boot:first",
		"boot:second",
	}

	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("unexpected module lifecycle: got %v, want %v", events, wantEvents)
	}

	if runtime.App() != app {
		t.Fatal("runtime contains a different app")
	}

	if runtime.Profile() != profile {
		t.Fatal("runtime contains a different profile")
	}

	if runtime.Registry() == nil {
		t.Fatal("runtime registry is nil")
	}

	if firstModule.registerRegistry == nil || secondModule.registerRegistry == nil {
		t.Fatal("module registry is nil")
	}

	if firstModule.registerRegistry == secondModule.registerRegistry {
		t.Fatal("modules received the same module-scoped registry")
	}

	if firstModule.bootContext.App() != app || secondModule.bootContext.App() != app {
		t.Fatal("module context contains a different app")
	}

	if firstModule.bootContext.Runtime() != runtime || secondModule.bootContext.Runtime() != runtime {
		t.Fatal("module context contains a different runtime")
	}

	if !reflect.DeepEqual(firstModule.bootContext.ModuleConfig(), firstConfig) {
		t.Fatal("first module received a different config")
	}

	if !reflect.DeepEqual(secondModule.bootContext.ModuleConfig(), secondConfig) {
		t.Fatal("second module received a different config")
	}
}

func TestSiteRuntimeFactory_MakeStopsWhenRegisterFails(t *testing.T) {
	app := kernel.NewApp(kernel.AppConfig{})
	events := make([]string, 0)
	registerErr := errors.New("register failed")

	firstModule := &recordingModule{
		code:   "first",
		events: &events,
	}

	secondModule := &recordingModule{
		code:        "second",
		events:      &events,
		registerErr: registerErr,
	}

	profile := &runtimeTestProfile{
		code: "test",
		modules: []kernel.ProfileModule{
			{Module: firstModule},
			{Module: secondModule},
		},
	}

	factory := kernel.NewSiteRuntimeFactory(app)
	runtime, err := factory.Make(context.Background(), profile)

	if runtime != nil {
		t.Fatal("expected nil runtime")
	}

	if !errors.Is(err, registerErr) {
		t.Fatalf("expected wrapped register error, got %v", err)
	}

	if !strings.Contains(err.Error(), `register module "second"`) {
		t.Fatalf("error does not contain module context: %v", err)
	}

	wantEvents := []string{
		"register:first",
		"register:second",
	}

	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("unexpected module lifecycle: got %v, want %v", events, wantEvents)
	}
}

func TestSiteRuntimeFactory_MakeStopsWhenBootFails(t *testing.T) {
	app := kernel.NewApp(kernel.AppConfig{})
	events := make([]string, 0)
	bootErr := errors.New("boot failed")

	firstModule := &recordingModule{
		code:    "first",
		events:  &events,
		bootErr: bootErr,
	}

	secondModule := &recordingModule{
		code:   "second",
		events: &events,
	}

	profile := &runtimeTestProfile{
		code: "test",
		modules: []kernel.ProfileModule{
			{Module: firstModule},
			{Module: secondModule},
		},
	}

	factory := kernel.NewSiteRuntimeFactory(app)
	runtime, err := factory.Make(context.Background(), profile)

	if runtime != nil {
		t.Fatal("expected nil runtime")
	}

	if !errors.Is(err, bootErr) {
		t.Fatalf("expected wrapped boot error, got %v", err)
	}

	if !strings.Contains(err.Error(), `boot module "first"`) {
		t.Fatalf("error does not contain module context: %v", err)
	}

	wantEvents := []string{
		"register:first",
		"register:second",
		"boot:first",
	}

	if !reflect.DeepEqual(events, wantEvents) {
		t.Fatalf("unexpected module lifecycle: got %v, want %v", events, wantEvents)
	}
}
