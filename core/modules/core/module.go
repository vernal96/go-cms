package coremodule

import (
	"context"

	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/core/modules/core/controllers"
	"github.com/vernal96/go-cms/core/modules/core/widgets"
)

const ModuleCode = "core"

type Config struct{}

type Module struct {
	config Config
}

func New(config Config) *Module {
	return &Module{
		config: config,
	}
}

func (m *Module) Code() string {
	return ModuleCode
}

func (m *Module) Register(registry core.Registry) error {
	return core.RegisterModule(registry.ForModule(m.Code()), core.ModuleRegistry{
		Widgets: []core.Widget{
			widgets.NewSiteInfoWidget(),
		},
		Controllers: []core.Controller{
			controllers.NewSiteController(),
		},
	})
}

func (m *Module) Boot(ctx context.Context, moduleContext core.ModuleContext) error {
	return ctx.Err()
}

var _ core.Module = (*Module)(nil)
