package coremodule

import (
	"context"
	"fmt"

	"github.com/vernal96/go-cms/core"
	"github.com/vernal96/go-cms/core/modules/core/widgets"
)

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
	return "core"
}

func (m *Module) Register(registry core.Registry) error {
	return core.RegisterModule(registry, core.ModuleRegistry{
		Widgets: []core.Widget{
			widgets.NewSiteInfoWidget(),
		},
	})
}

func (m *Module) Boot(ctx context.Context, moduleContext core.ModuleContext) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	site := moduleContext.Runtime().Site()

	fmt.Println("core module booted for site:", site.Domain)

	return nil
}

var _ core.Module = (*Module)(nil)
