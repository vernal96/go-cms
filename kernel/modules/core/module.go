package core

import "github.com/vernal96/go-cms/kernel"

type Module struct{}

func (m Module) Code() kernel.ModuleCode {
	return "core"
}
