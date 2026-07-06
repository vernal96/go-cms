package core

import "context"

type ModuleCode string

type Module interface {
	Code() ModuleCode
	Register(registry Registry) error
	Boot(ctx context.Context, app *App) error
}
