package core

import (
	"context"
	"fmt"

	"github.com/vernal96/go-cms/kernel"
	"github.com/vernal96/go-cms/kernel/modules/core/site"
)

const ModuleCode kernel.ModuleCode = "core"

type Module struct{}

func (m Module) Code() kernel.ModuleCode {
	return ModuleCode
}

func (m Module) Register(registry kernel.Registry) error {
	return nil
}

func (m Module) Boot(ctx context.Context, moduleContext kernel.ModuleContext) error {
	moduleConfig, err := kernel.ModuleConfigFrom[Config](moduleContext)
	if err != nil {
		return err
	}

	if moduleConfig.Site.RepositoryAdapter == "" {
		return fmt.Errorf("core site repository adapter is not configured")
	}

	siteRepository, err := kernel.AdapterAs[site.Repository](
		moduleContext.App().Adapters(),
		site.RepositoryAdapterContract,
		moduleConfig.Site.RepositoryAdapter,
	)
	if err != nil {
		return err
	}

	_ = siteRepository

	return ctx.Err()
}

var _ kernel.Module = Module{}
