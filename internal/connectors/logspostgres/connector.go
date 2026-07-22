package logspostgres

import (
	"context"
	"time"

	"github.com/kelseyhightower/envconfig"
	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel"
)

const ConnectionCode kernel.ConnectionCode = "logs"

// Config intentionally has its own environment names even though it reuses
// the same PostgreSQL connector implementation as the main connection.
type Config struct {
	Host            string        `envconfig:"LOGS_POSTGRES_HOST" required:"true"`
	Port            int           `envconfig:"LOGS_POSTGRES_PORT" required:"true"`
	Database        string        `envconfig:"LOGS_POSTGRES_DB" required:"true"`
	User            string        `envconfig:"LOGS_POSTGRES_USER" required:"true"`
	Password        string        `envconfig:"LOGS_POSTGRES_PASSWORD" required:"true"`
	SSLMode         string        `envconfig:"LOGS_POSTGRES_SSL_MODE" default:"disable"`
	MaxOpenConns    int32         `envconfig:"LOGS_POSTGRES_MAX_OPEN_CONNS" default:"10"`
	MinConns        int32         `envconfig:"LOGS_POSTGRES_MIN_CONNS" default:"0"`
	ConnMaxLifetime time.Duration `envconfig:"LOGS_POSTGRES_CONN_MAX_LIFETIME" default:"30m"`
	ConnectTimeout  time.Duration `envconfig:"LOGS_POSTGRES_CONNECT_TIMEOUT" default:"5s"`
}

func Load() (*Config, error) {
	var config Config

	if err := envconfig.Process("", &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func New(
	ctx context.Context,
	config Config,
) (*connectorpostgres.Connector, error) {
	return connectorpostgres.New(ctx, connectorpostgres.Config{
		Code:            ConnectionCode,
		Host:            config.Host,
		Port:            config.Port,
		Database:        config.Database,
		User:            config.User,
		Password:        config.Password,
		SSLMode:         config.SSLMode,
		MaxConns:        config.MaxOpenConns,
		MinConns:        config.MinConns,
		ConnMaxLifetime: config.ConnMaxLifetime,
		ConnectTimeout:  config.ConnectTimeout,
	})
}
