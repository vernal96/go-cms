package mainpostgres

import (
	"time"

	connectorpostgres "github.com/vernal96/go-cms/connectors/postgres"
	"github.com/vernal96/go-cms/kernel"
)

const ConnectionCode kernel.ConnectionCode = "main"

type Config struct {
	Host            string        `envconfig:"HOST" required:"true"`
	Port            int           `envconfig:"PORT" required:"true"`
	Database        string        `envconfig:"DB" required:"true"`
	User            string        `envconfig:"USER" required:"true"`
	Password        string        `envconfig:"PASSWORD" required:"true"`
	SSLMode         string        `envconfig:"SSL_MODE" default:"disable"`
	MaxOpenConns    int32         `envconfig:"MAX_OPEN_CONNS" default:"10"`
	MinConns        int32         `envconfig:"MIN_CONNS" default:"0"`
	ConnMaxLifetime time.Duration `envconfig:"CONN_MAX_LIFETIME" default:"30m"`
	ConnectTimeout  time.Duration `envconfig:"CONNECT_TIMEOUT" default:"5s"`
}

func Factory(config Config) connectorpostgres.Factory {
	return connectorpostgres.Factory{
		Config: connectorpostgres.Config{
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
		},
	}
}
