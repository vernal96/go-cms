package core

import "context"

type Module interface {
	Code() string
	Register() error
	Boot(ctx context.Context, app *App) error
}
