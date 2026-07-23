package core

import (
	"context"
	"errors"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core/field"
	"github.com/vernal96/go-cms/kernel/modules/core/resource"
	"github.com/vernal96/go-cms/kernel/modules/core/resourcetype"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

const ModuleCode kernel.ModuleCode = "core"

// Database is the persistence boundary required by the core module.
// Its concrete implementation is selected by the main application binding.
type Database interface {
	kernel.ModuleDatabase
	Sites() site.Repository
	Resources() resource.Repository
}

type Module struct{}

func (Module) Code() kernel.ModuleCode {
	return ModuleCode
}

func (Module) Registry() kernel.ModuleRegistry {
	return kernel.ModuleRegistry{
		FieldTypes:    field.StandardTypes(),
		ResourceTypes: resourcetype.StandardTypes(),
	}
}

func (Module) Build(
	_ context.Context,
	ctx kernel.ModuleContext,
) (kernel.ModuleRuntime, error) {
	database, err := kernel.ModuleDatabaseFrom[Database](
		ctx,
		"",
		ModuleCode,
	)
	if err != nil {
		return nil, err
	}

	if database.Sites() == nil {
		return nil, errors.New("core site repository is nil")
	}
	if database.Resources() == nil {
		return nil, errors.New("core resource repository is nil")
	}

	return &Runtime{database: database}, nil
}

type Runtime struct {
	database Database
}

func (r *Runtime) ModuleCode() kernel.ModuleCode {
	return ModuleCode
}

func (r *Runtime) Database() Database {
	return r.database
}

var _ kernel.Module = Module{}
var _ kernel.RegistryProvider = Module{}
var _ kernel.ModuleRuntime = (*Runtime)(nil)
