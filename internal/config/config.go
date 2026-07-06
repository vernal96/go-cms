package config

import (
	"net"
	"strconv"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Server ServerConfig
}

type ServerConfig struct {
	Host            string        `envconfig:"SERVER_HOST" default:""`
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
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
