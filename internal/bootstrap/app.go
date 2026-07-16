package bootstrap

import (
	"context"
	"fmt"

	"github.com/vernal96/go-cms/internal/config"
	"github.com/vernal96/go-cms/kernel"

	connectormemory "github.com/vernal96/go-cms/connectors/memory"
)

func NewApp(
	ctx context.Context,
	cfg *config.Config,
) (_ *kernel.App, err error) {
	if cfg == nil {
		return nil, fmt.Errorf("project config is nil")
	}

	mainDB := connectormemory.New()

	defer func() {
		if err != nil {
			_ = mainDB.Close()
		}
	}()

	coreDatabase, err :=
}
