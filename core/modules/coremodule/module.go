package coremodule

import "github.com/vernal96/go-cms/core"

type Module struct{}

func (m Module) Code() core.ModuleCode {
	return "core"
}
