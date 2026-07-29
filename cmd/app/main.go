package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	projectconfig "github.com/vernal96/go-cms/internal/config"
	jwtsecurity "github.com/vernal96/go-cms/internal/security/jwt"
	httpserver "github.com/vernal96/go-cms/internal/server/http"
	appkernel "github.com/vernal96/go-cms/kernel/app"
	"github.com/vernal96/go-cms/kernel/logging"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	if err := run(ctx); err != nil {
		if !logging.IsReported(err) {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func run(ctx context.Context) (resultErr error) {
	projectConfig, err := projectconfig.Load()
	if err != nil {
		return err
	}

	application, err := appkernel.New(ctx, projectConfig.Application())
	if err != nil {
		return err
	}
	defer func() {
		if resultErr != nil {
			application.Logger().Error(
				"web process failed",
				slog.String("event", "process.web.failed"),
				slog.Any("error", resultErr),
			)
		}
		resultErr = errors.Join(resultErr, application.Close())
		if resultErr != nil {
			resultErr = logging.Reported(resultErr)
		}
	}()
	slog.SetDefault(application.Logger())

	accessTokens, err := jwtsecurity.New(projectConfig.JWT)
	if err != nil {
		application.Logger().ErrorContext(
			ctx,
			"initialize access tokens",
			slog.String("event", "app.security.initialization.failed"),
			slog.Any("error", err),
		)
		return err
	}

	if err := application.Boot(ctx); err != nil {
		return err
	}

	handler, err := httpserver.NewHandler(
		application,
		httpserver.WithAccessTokens(accessTokens),
	)
	if err != nil {
		application.Logger().ErrorContext(
			ctx,
			"initialize HTTP handler",
			slog.String("event", "http.handler.initialization.failed"),
			slog.Any("error", err),
		)
		return err
	}

	server, err := httpserver.NewServer(
		projectConfig.Server,
		handler,
		application.Logger(),
	)
	if err != nil {
		return err
	}
	return server.Run(ctx)
}
