package http

import (
	"context"
	"errors"
	stdhttp "net/http"

	"github.com/vernal96/go-cms/internal/config"
)

type Server struct {
	server *stdhttp.Server
	cfg    config.ServerConfig
}

func NewServer(cfg config.ServerConfig, handler stdhttp.Handler) *Server {
	return &Server{
		server: &stdhttp.Server{
			Addr:         cfg.Address(),
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
		cfg: cfg,
	}
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		err := s.server.ListenAndServe()
		if err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			errCh <- err
			return
		}

		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()

		return s.server.Shutdown(shutdownCtx)

	case err := <-errCh:
		return err
	}
}
