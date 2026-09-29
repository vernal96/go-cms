package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/vernal96/go-cms-kernel/console"
	"github.com/vernal96/go-cms/internal/bootstrap"
	"github.com/vernal96/go-cms/internal/config"
	"github.com/vernal96/go-cms/internal/server"
)

//go:embed seeds/*.sql
var seedFiles embed.FS

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		// The project logger writes to a file; failures must also reach stderr.
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) (resultErr error) {
	mode, err := executableMode(args)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	var httpConfig config.HTTPConfig
	if mode == bootstrap.HTTP {
		httpConfig, err = config.LoadHTTP()
		if err != nil {
			return err
		}
	}
	application, err := bootstrap.New(ctx, cfg, seedFiles, mode)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, application.Close()) }()
	if mode == bootstrap.Console {
		return application.Console().Run(ctx, args[1:], console.StandardIO())
	}
	return server.Run(ctx, application, httpConfig)
}

func executableMode(args []string) (bootstrap.Mode, error) {
	if len(args) == 0 {
		return bootstrap.HTTP, nil
	}
	if args[0] == "console" {
		return bootstrap.Console, nil
	}
	return 0, errors.New("usage: server [console [command [arguments]]]")
}
