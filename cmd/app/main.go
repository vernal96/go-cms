package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	"github.com/vernal96/go-cms/internal/bootstrap"
	"github.com/vernal96/go-cms/internal/config"
	httpserver "github.com/vernal96/go-cms/internal/server/http"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	projectConfig, err := config.Load()
	if err != nil {
		panic(err)
	}

	app, err := bootstrap.NewApp(ctx, projectConfig)
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := app.Close(); closeErr != nil {
			panic(errors.Join(err, closeErr))
		}
	}()

	handler, err := httpserver.NewHandler(app)
	if err != nil {
		panic(err)
	}

	server := httpserver.NewServer(
		projectConfig.Server,
		handler,
	)

	if err := server.Run(ctx); err != nil {
		panic(err)
	}
}
