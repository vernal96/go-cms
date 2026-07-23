package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	projectconfig "github.com/vernal96/go-cms/internal/config"
	httpserver "github.com/vernal96/go-cms/internal/server/http"
	appkernel "github.com/vernal96/go-cms/kernel/app"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	projectConfig, err := projectconfig.Load()
	if err != nil {
		panic(err)
	}

	application, err := appkernel.New(ctx, projectConfig.Application())
	if err != nil {
		panic(err)
	}
	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			panic(errors.Join(err, closeErr))
		}
	}()

	if err := application.Boot(ctx); err != nil {
		panic(err)
	}

	handler, err := httpserver.NewHandler(application)
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
