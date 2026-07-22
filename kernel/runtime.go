package kernel

import (
	"context"
	"errors"
	"fmt"
)

type ModuleCode string
type ProfileCode string

type Profile struct {
	Code    ProfileCode
	Modules []ProfileModule
}

type ProfileModule struct {
	Module Module
	Config any
}

type Module interface {
	Code() ModuleCode
	Build(context.Context, ModuleContext) (ModuleRuntime, error)
}

type ModuleRuntime interface {
	ModuleCode() ModuleCode
}

type Registry interface {
	Module(ModuleCode) (ModuleRuntime, bool)
}

type RuntimeRegistry struct {
	modules map[ModuleCode]ModuleRuntime
}

func newRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{
		modules: make(map[ModuleCode]ModuleRuntime),
	}
}

func (r *RuntimeRegistry) Module(
	code ModuleCode,
) (ModuleRuntime, bool) {
	runtime, exists := r.modules[code]
	return runtime, exists
}

func (r *RuntimeRegistry) add(runtime ModuleRuntime) error {
	if runtime == nil {
		return errors.New("module runtime is nil")
	}

	code := runtime.ModuleCode()
	if code == "" {
		return errors.New("module runtime code is empty")
	}

	if _, exists := r.modules[code]; exists {
		return fmt.Errorf(
			"module runtime %q already exists",
			code,
		)
	}

	r.modules[code] = runtime
	return nil
}

type ModuleContext struct {
	resolver DatabaseResolver
	profile  Profile
	registry Registry
	config   any
}

func newModuleContext(
	resolver DatabaseResolver,
	profile Profile,
	registry Registry,
	config any,
) ModuleContext {
	return ModuleContext{
		resolver: resolver,
		profile:  cloneProfile(profile),
		registry: registry,
		config:   config,
	}
}

func (c ModuleContext) Profile() Profile {
	return cloneProfile(c.profile)
}

func (c ModuleContext) Registry() Registry {
	return c.registry
}

func (c ModuleContext) Config() any {
	return c.config
}

func ModuleConfigFrom[T any](ctx ModuleContext) (T, error) {
	var zero T

	if ctx.config == nil {
		return zero, nil
	}

	if config, ok := ctx.config.(T); ok {
		return config, nil
	}

	if config, ok := ctx.config.(*T); ok {
		if config == nil {
			return zero, errors.New("module config is nil")
		}

		return *config, nil
	}

	return zero, fmt.Errorf(
		"invalid module config type %T, expected %T",
		ctx.config,
		zero,
	)
}

func ModuleDatabaseFrom[T ModuleDatabase](
	ctx ModuleContext,
	connectionCode ConnectionCode,
	moduleCode ModuleCode,
) (T, error) {
	var zero T

	var (
		database ModuleDatabase
		exists   bool
	)

	if connectionCode == "" {
		database, exists = ctx.resolver.MainModuleDatabase(
			moduleCode,
		)
	} else {
		database, exists = ctx.resolver.ModuleDatabase(
			connectionCode,
			moduleCode,
		)
	}

	if !exists {
		return zero, fmt.Errorf(
			"database for module %q on connection %q not found",
			moduleCode,
			connectionCode,
		)
	}

	result, ok := database.(T)
	if !ok {
		return zero, fmt.Errorf(
			"database for module %q has invalid type %T",
			moduleCode,
			database,
		)
	}

	return result, nil
}

type ProfileRuntime struct {
	profile  Profile
	registry Registry
}

func newProfileRuntime(
	profile Profile,
	registry Registry,
) *ProfileRuntime {
	return &ProfileRuntime{
		profile:  cloneProfile(profile),
		registry: registry,
	}
}

func (r *ProfileRuntime) Profile() Profile {
	return cloneProfile(r.profile)
}

func (r *ProfileRuntime) Registry() Registry {
	return r.registry
}

type ProfileRuntimeFactory struct {
	resolver DatabaseResolver
}

func NewProfileRuntimeFactory(
	resolver DatabaseResolver,
) (*ProfileRuntimeFactory, error) {
	if resolver == nil {
		return nil, errors.New("database resolver is nil")
	}

	return &ProfileRuntimeFactory{
		resolver: resolver,
	}, nil
}

func (f *ProfileRuntimeFactory) Make(
	ctx context.Context,
	profile Profile,
) (*ProfileRuntime, error) {
	if ctx == nil {
		return nil, errors.New("profile runtime context is nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if profile.Code == "" {
		return nil, errors.New("profile code is empty")
	}

	profile = cloneProfile(profile)

	moduleCodes := make(
		map[ModuleCode]struct{},
		len(profile.Modules),
	)

	for index, profileModule := range profile.Modules {
		if profileModule.Module == nil {
			return nil, fmt.Errorf(
				"profile %q module at index %d is nil",
				profile.Code,
				index,
			)
		}

		moduleCode := profileModule.Module.Code()
		if moduleCode == "" {
			return nil, fmt.Errorf(
				"profile %q module at index %d has empty code",
				profile.Code,
				index,
			)
		}

		if _, exists := moduleCodes[moduleCode]; exists {
			return nil, fmt.Errorf(
				"profile %q contains duplicate module %q",
				profile.Code,
				moduleCode,
			)
		}

		moduleCodes[moduleCode] = struct{}{}
	}

	registry := newRuntimeRegistry()

	for _, profileModule := range profile.Modules {
		module := profileModule.Module

		moduleContext := newModuleContext(
			f.resolver,
			profile,
			registry,
			profileModule.Config,
		)

		runtime, err := module.Build(ctx, moduleContext)
		if err != nil {
			return nil, fmt.Errorf(
				"build module %q for profile %q: %w",
				module.Code(),
				profile.Code,
				err,
			)
		}

		if runtime == nil {
			return nil, fmt.Errorf(
				"module %q returned nil runtime",
				module.Code(),
			)
		}

		if runtime.ModuleCode() != module.Code() {
			return nil, fmt.Errorf(
				"module %q returned runtime for module %q",
				module.Code(),
				runtime.ModuleCode(),
			)
		}

		if err := registry.add(runtime); err != nil {
			return nil, err
		}
	}

	return newProfileRuntime(profile, registry), nil
}

func cloneProfile(profile Profile) Profile {
	profile.Modules = append(
		[]ProfileModule(nil),
		profile.Modules...,
	)

	return profile
}
