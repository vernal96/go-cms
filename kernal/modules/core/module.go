package core

import "github.com/vernal96/go-cms/kernal"

type Module struct{}

func (m Module) Code() kernal.ModuleCode {
	return "core"
}
