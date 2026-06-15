package core

import "context"

type Module interface {
	Code() string
	Register(registry Registry) error
	Boot(ctx context.Context, moduleContext ModuleContext) error
}
