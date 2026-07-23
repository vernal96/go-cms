package kernel

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/file"
	"github.com/vernal96/go-cms/kernel/modules/core/group"
	"github.com/vernal96/go-cms/kernel/modules/core/media"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/template"
	"github.com/vernal96/go-cms/kernel/modules/core/user"
	"github.com/vernal96/go-cms/kernel/permission"
	"github.com/vernal96/go-cms/kernel/security"
)

type ModuleCode string
type ProfileCode string

type Profile struct {
	Code      ProfileCode
	Modules   []ProfileModule
	Params    []field.Definition
	Templates []template.Definition
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
	ResourceType(resourcetype.Code) (resourcetype.Type, bool)
	Permission(permission.Code) (permission.Definition, bool)
	Permissions() []permission.Code
}

type ModuleRegistry struct {
	FieldTypes         []field.Type
	ResourceTypes      []resourcetype.Type
	PermissionEntities []permission.Entity
}

type RegistryProvider interface {
	Registry() ModuleRegistry
}

type RuntimeRegistry struct {
	modules         map[ModuleCode]ModuleRuntime
	fieldTypes      map[field.TypeCode]field.Type
	resourceTypes   map[resourcetype.Code]resourcetype.Type
	permissions     map[permission.Code]permission.Definition
	permissionCodes []permission.Code
}

func newRuntimeRegistry() *RuntimeRegistry {
	return &RuntimeRegistry{
		modules:    make(map[ModuleCode]ModuleRuntime),
		fieldTypes: make(map[field.TypeCode]field.Type),
		resourceTypes: make(
			map[resourcetype.Code]resourcetype.Type,
		),
		permissions: make(
			map[permission.Code]permission.Definition,
		),
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

func (r *RuntimeRegistry) ResourceType(
	code resourcetype.Code,
) (resourcetype.Type, bool) {
	resourceType, exists := r.resourceTypes[code]
	return resourceType, exists
}

func (r *RuntimeRegistry) Permission(
	code permission.Code,
) (permission.Definition, bool) {
	definition, exists := r.permissions[code]
	return definition, exists
}

func (r *RuntimeRegistry) Permissions() []permission.Code {
	return append([]permission.Code(nil), r.permissionCodes...)
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

func (r *RuntimeRegistry) addResourceType(
	resourceType resourcetype.Type,
) error {
	if resourceType == nil || isNilValue(resourceType) {
		return errors.New("resource type is nil")
	}

	code := resourceType.Code()
	if code == "" {
		return errors.New("resource type code is empty")
	}
	if _, exists := r.resourceTypes[code]; exists {
		return fmt.Errorf("resource type %q already exists", code)
	}

	switch resourceType.PathMode() {
	case resourcetype.PathRoute, resourcetype.PathNone:
	default:
		return fmt.Errorf(
			"resource type %q has invalid path mode %q",
			code,
			resourceType.PathMode(),
		)
	}

	r.resourceTypes[code] = resourceType
	return nil
}

func (r *RuntimeRegistry) addPermission(
	definition permission.Definition,
) error {
	if definition.Code == "" {
		return errors.New("permission code is empty")
	}
	if _, exists := r.permissions[definition.Code]; exists {
		return fmt.Errorf(
			"permission %q already exists",
			definition.Code,
		)
	}
	r.permissions[definition.Code] = definition
	r.permissionCodes = append(
		r.permissionCodes,
		definition.Code,
	)
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
	resolver      DatabaseResolver
	profile       Profile
	registry      Registry
	config        any
	files         file.Service
	media         media.Service
	users         user.Service
	groups        group.Service
	authorization security.Authorizer
}

func newModuleContext(
	resolver DatabaseResolver,
	profile Profile,
	registry Registry,
	config any,
	services RuntimeServices,
) ModuleContext {
	return ModuleContext{
		resolver:      resolver,
		profile:       cloneProfile(profile),
		registry:      registry,
		config:        config,
		files:         services.Files,
		media:         services.Media,
		users:         services.Users,
		groups:        services.Groups,
		authorization: services.Authorization,
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

// Files exposes the core application file service without exposing the
// filesystem manager or any infrastructure connector to modules.
func (c ModuleContext) Files() file.Service {
	return c.files
}

// Media exposes the core application media service without exposing its
// persistence adapter or the filesystem manager to modules.
func (c ModuleContext) Media() media.Service {
	return c.media
}

func (c ModuleContext) Users() user.Service {
	return c.users
}

func (c ModuleContext) Groups() group.Service {
	return c.groups
}

func (c ModuleContext) Authorization() security.Authorizer {
	return c.authorization
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
	templates   *template.Catalog
}

func newProfileRuntime(
	profile Profile,
	registry Registry,
	paramSchema *field.Schema,
	templates *template.Catalog,
) *ProfileRuntime {
	return &ProfileRuntime{
		profile:     cloneProfile(profile),
		registry:    registry,
		paramSchema: paramSchema,
		templates:   templates,
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

func (r *ProfileRuntime) Template(
	code template.Code,
) (*template.Runtime, bool) {
	if r == nil || r.templates == nil {
		return nil, false
	}

	return r.templates.Template(code)
}

func (r *ProfileRuntime) Templates() []template.Definition {
	if r == nil || r.templates == nil {
		return nil
	}

	return r.templates.Definitions()
}

type ProfileRuntimeFactory struct {
	resolver DatabaseResolver
	services RuntimeServices
}

type RuntimeServices struct {
	Files         file.Service
	Media         media.Service
	Users         user.Service
	Groups        group.Service
	Authorization security.Authorizer
}

func NewProfileRuntimeFactory(
	resolver DatabaseResolver,
	services ...RuntimeServices,
) (*ProfileRuntimeFactory, error) {
	if resolver == nil {
		return nil, errors.New("database resolver is nil")
	}

	factory := &ProfileRuntimeFactory{resolver: resolver}
	if len(services) > 1 {
		return nil, errors.New(
			"more than one runtime services set was provided",
		)
	}
	if len(services) == 1 {
		factory.services = services[0]
	}
	return factory, nil
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
		moduleRegistry := provider.Registry()

		for index, fieldType := range moduleRegistry.FieldTypes {
			if err := registry.addFieldType(fieldType); err != nil {
				return nil, fmt.Errorf(
					"register field type at index %d from module %q: %w",
					index,
					profileModule.Module.Code(),
					err,
				)
			}
		}
		for index, resourceType := range moduleRegistry.ResourceTypes {
			if err := registry.addResourceType(resourceType); err != nil {
				return nil, fmt.Errorf(
					"register resource type at index %d from module %q: %w",
					index,
					profileModule.Module.Code(),
					err,
				)
			}
		}
		permissionDefinitions, err := permission.Definitions(
			string(profileModule.Module.Code()),
			moduleRegistry.PermissionEntities,
		)
		if err != nil {
			return nil, fmt.Errorf(
				"register permissions from module %q: %w",
				profileModule.Module.Code(),
				err,
			)
		}
		for index, definition := range permissionDefinitions {
			if err := registry.addPermission(definition); err != nil {
				return nil, fmt.Errorf(
					"register permission at index %d from module %q: %w",
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

	templates, err := template.Compile(
		profile.Templates,
		registry,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"compile templates for profile %q: %w",
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
			f.services,
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

	return newProfileRuntime(
		profile,
		registry,
		paramSchema,
		templates,
	), nil
}

func cloneProfile(profile Profile) Profile {
	profile.Modules = append(
		[]ProfileModule(nil),
		profile.Modules...,
	)
	profile.Params = field.CloneDefinitions(profile.Params)
	profile.Templates = template.CloneDefinitions(profile.Templates)

	return profile
}
