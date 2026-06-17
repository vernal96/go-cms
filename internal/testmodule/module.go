package testmodule

import (
	"context"
	"errors"
	"fmt"

	"github.com/vernal96/go-cms/core"
)

type Config struct {
	CacheStore core.CacheStoreName
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
	if m.config.CacheStore == "" {
		return errors.New("test module cache store is empty")
	}

	if m.config.FileDisk == "" {
		return errors.New("test module file disk is empty")
	}

	cacheStore, err := moduleContext.App().CacheManager().Store(m.config.CacheStore)
	if err != nil {
		return fmt.Errorf("get test module cache store %q: %w", m.config.CacheStore, err)
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
