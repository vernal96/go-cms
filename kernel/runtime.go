package kernel

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
)

type ModuleCode string
type ProfileCode string

type Profile struct {
	Code    ProfileCode
	Modules []ProfileModule
	Params  []field.Definition
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
	FieldType(field.TypeCode) (field.Type, bool)
}

type ModuleRegistry struct {
	FieldTypes []field.Type
}

type RegistryProvider interface {
	Registry() ModuleRegistry
}

type RuntimeRegistry struct {
	modules    map[ModuleCode]ModuleRuntime
	fieldTypes map[field.TypeCode]field.Type
}

func newRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{
		modules:    make(map[ModuleCode]ModuleRuntime),
		fieldTypes: make(map[field.TypeCode]field.Type),
	}
}

func (r *RuntimeRegistry) Module(
	code ModuleCode,
) (ModuleRuntime, bool) {
	runtime, exists := r.modules[code]
	return runtime, exists
}

func (r *RuntimeRegistry) FieldType(
	code field.TypeCode,
) (field.Type, bool) {
	fieldType, exists := r.fieldTypes[code]
	return fieldType, exists
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

func (r *RuntimeRegistry) addFieldType(
	fieldType field.Type,
) error {
	if fieldType == nil || isNilValue(fieldType) {
		return errors.New("field type is nil")
	}

	code := fieldType.Code()
	if code == "" {
		return errors.New("field type code is empty")
	}
	if _, exists := r.fieldTypes[code]; exists {
		return fmt.Errorf("field type %q already exists", code)
	}

	r.fieldTypes[code] = fieldType
	return nil
}

func isNilValue(value any) bool {
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
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
	profile     Profile
	registry    Registry
	paramSchema *field.Schema
}

func newProfileRuntime(
	profile Profile,
	registry Registry,
	paramSchema *field.Schema,
) *ProfileRuntime {
	return &ProfileRuntime{
		profile:     cloneProfile(profile),
		registry:    registry,
		paramSchema: paramSchema,
	}
}

func (r *ProfileRuntime) Profile() Profile {
	return cloneProfile(r.profile)
}

func (r *ProfileRuntime) Registry() Registry {
	return r.registry
}

func (r *ProfileRuntime) ParamSchema() *field.Schema {
	return r.paramSchema
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
		provider, ok := profileModule.Module.(RegistryProvider)
		if !ok {
			continue
		}

		for index, fieldType := range provider.Registry().FieldTypes {
			if err := registry.addFieldType(fieldType); err != nil {
				return nil, fmt.Errorf(
					"register field type at index %d from module %q: %w",
					index,
					profileModule.Module.Code(),
					err,
				)
			}
		}
	}

	paramSchema, err := field.Compile(profile.Params, registry)
	if err != nil {
		return nil, fmt.Errorf(
			"compile params for profile %q: %w",
			profile.Code,
			err,
		)
	}

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

	return newProfileRuntime(profile, registry, paramSchema), nil
}

func cloneProfile(profile Profile) Profile {
	profile.Modules = append(
		[]ProfileModule(nil),
		profile.Modules...,
	)
	profile.Params = field.CloneDefinitions(profile.Params)

	return profile
}
