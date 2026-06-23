package core

import "context"

const CoreModuleCode = "core"

type CoreModule struct{}

func NewCoreModule() *CoreModule {
	return &CoreModule{}
}

func (m *CoreModule) Code() string {
	return CoreModuleCode
}

func (m *CoreModule) Register(registry Registry) error {
	registry = registry.ForModule(m.Code())

	if err := registry.Widgets().Register(NewSiteInfoWidget()); err != nil {
		return err
	}

	if err := registry.Controllers().Register(NewSiteController()); err != nil {
		return err
	}

	return nil
}

func (m *CoreModule) Boot(ctx context.Context, moduleContext ModuleContext) error {
	return ctx.Err()
}

var _ Module = (*CoreModule)(nil)
