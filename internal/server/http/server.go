package httpserver

import (
	"context"
	"errors"
	"net/http"

	"github.com/vernal96/go-cms/internal/config"
)

type Server struct {
	server *http.Server
	config config.ServerConfig
}

func NewServer(
	config config.ServerConfig,
	handler http.Handler,
) *Server {
	return &Server{
		server: &http.Server{
			Addr:         config.Address(),
			Handler:      handler,
			ReadTimeout:  config.ReadTimeout,
			WriteTimeout: config.WriteTimeout,
			IdleTimeout:  config.IdleTimeout,
		},
		config: config,
	}
}

func (s *Server) Run(ctx context.Context) error {
	result := make(chan error, 1)

	go func() {
		err := s.server.ListenAndServe()

		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}

		result <- err
	}()

	select {
	case err := <-result:
		return err

	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(
			context.Background(),
			s.config.ShutdownTimeout,
		)
		defer cancel()

		return s.server.Shutdown(shutdownContext)
	}
}
