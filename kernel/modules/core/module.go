package core

import (
	"context"

	"github.com/vernal96/go-cms/kernel"
)

const ModuleCode kernel.ModuleCode = "core"

type Module struct{}

func (m Module) Code() kernel.ModuleCode {
	return ModuleCode
}

func (m Module) Register(registry kernel.Registry) error {
	return nil
}

func (m Module) Boot(ctx context.Context, moduleContext kernel.ModuleContext) error {

	return ctx.Err()
}

var _ kernel.Module = Module{}
