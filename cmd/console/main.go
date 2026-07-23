package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	projectconfig "github.com/vernal96/go-cms/internal/config"
	appkernel "github.com/vernal96/go-cms/kernel/app"
	"github.com/vernal96/go-cms/kernel/console"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	if err := run(ctx, os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) (resultErr error) {
	projectConfig, err := projectconfig.Load()
	if err != nil {
		return err
	}

	application, err := appkernel.New(ctx, projectConfig.Application())
	if err != nil {
		return err
	}
	defer func() {
		resultErr = errors.Join(resultErr, application.Close())
	}()

	return application.Console().Run(ctx, args, console.StandardIO())
}
