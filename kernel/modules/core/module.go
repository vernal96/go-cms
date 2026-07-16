package core

import "github.com/vernal96/go-cms/kernel"

const ModuleCode kernel.ModuleCode = "core"

type Module struct {
	Config Config
}

func (m Module) Code() kernel.ModuleCode {
	return ModuleCode
}

var _ kernel.Module = (*Module)(nil)
