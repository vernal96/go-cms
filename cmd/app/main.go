package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/vernal96/go-cms/internal/bootstrap"
	"github.com/vernal96/go-cms/internal/config"
	serverhttp "github.com/vernal96/go-cms/internal/server/http"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	app, err := bootstrap.NewApp()
	if err != nil {
		panic(err)
	}

	profiles, err := bootstrap.NewProfileRegistry()
	if err != nil {
		panic(err)
	}

	runtimeResolver, err := bootstrap.NewRuntimeResolver(app, profiles, cfg)
	if err != nil {
		panic(err)
	}

	handler, err := serverhttp.NewHandler(runtimeResolver)
	if err != nil {
		panic(err)
	}

	server := serverhttp.NewServer(cfg.Server, handler)

	if err := server.Run(ctx); err != nil {
		panic(err)
	}
}
