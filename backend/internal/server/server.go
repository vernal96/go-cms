// Package server configures and runs the project's HTTP executable mode.
package server

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/vernal96/go-cms-kernel/app"
	"github.com/vernal96/go-cms-kernel/security/jwt"
	"github.com/vernal96/go-cms-kernel/transport/httpserver"
	"github.com/vernal96/go-cms/internal/config"
)

func Run(ctx context.Context, application *app.App, cfg config.HTTPConfig) error {
	accessTokens, err := jwt.New(cfg.JWT, jwt.WithSessions(application.Services().Sessions))
	if err != nil {
		return err
	}
	handler, err := httpserver.NewHandler(application, httpserver.WithAccessTokens(accessTokens))
	if err != nil {
		return err
	}
	server, err := httpserver.NewServer(httpserver.Config{
		Address:         net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port)),
		ReadTimeout:     cfg.Server.ReadTimeout,
		WriteTimeout:    cfg.Server.WriteTimeout,
		ShutdownTimeout: cfg.Server.ShutdownTimeout,
	}, rootHandler(handler), application.Logger())
	if err != nil {
		return err
	}
	return server.Run(ctx)
}

func rootHandler(handler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", handler)
	return mux
}
