package testmodule

import (
	"context"
	"errors"
	"fmt"

	"github.com/vernal96/go-cms/core"
)

const CacheScopeDefault core.CacheScope = "test.default"

type Config struct {
	CacheScope core.CacheScope
	FileDisk   core.FileDisk
}

type Module struct {
	config Config
}

func New(config Config) *Module {
	return &Module{
		config: config,
	}
}

func (m *Module) Code() string {
	return "test"
}

func (m *Module) Register(registry core.Registry) error {
	fmt.Println("test module registered")
	return nil
}

func (m *Module) Boot(ctx context.Context, moduleContext core.ModuleContext) error {
	if m.config.CacheScope == "" {
		return errors.New("test module cache scope is empty")
	}

	if m.config.FileDisk == "" {
		return errors.New("test module file disk is empty")
	}

	cacheStore, err := moduleContext.App().CacheManager().Scope(m.config.CacheScope)
	if err != nil {
		return fmt.Errorf("get test module cache scope %q: %w", m.config.CacheScope, err)
	}

	fileStorage, err := moduleContext.App().Storage().Disk(m.config.FileDisk)
	if err != nil {
		return fmt.Errorf("get test module file disk %q: %w", m.config.FileDisk, err)
	}

	runtime := moduleContext.Runtime()

	_ = cacheStore
	_ = fileStorage
	_ = runtime

	fmt.Println("test module booted")

	return nil
}
