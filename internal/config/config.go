package config

import (
	"net"
	"strconv"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/vernal96/go-cms/internal/connectors/mainpostgres"
)

type Config struct {
	Server   ServerConfig
	Postgres mainpostgres.Config
}

type ServerConfig struct {
	Host            string        `envconfig:"SERVER_HOST" default:"localhost"`
	Port            int           `envconfig:"SERVER_PORT" default:"8080"`
	ReadTimeout     time.Duration `envconfig:"SERVER_READ_TIMEOUT" default:"5s"`
	WriteTimeout    time.Duration `envconfig:"SERVER_WRITE_TIMEOUT" default:"10s"`
	IdleTimeout     time.Duration `envconfig:"SERVER_IDLE_TIMEOUT" default:"120s"`
	ShutdownTimeout time.Duration `envconfig:"SERVER_SHUTDOWN_TIMEOUT" default:"5s"`
}

func (c ServerConfig) Address() string {
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func Load() (*Config, error) {
	var server ServerConfig

	if err := envconfig.Process("", &server); err != nil {
		return nil, err
	}

	postgres, err := mainpostgres.Load()
	if err != nil {
		return nil, err
	}

	return &Config{
		Server:   server,
		Postgres: *postgres,
	}, nil
}
