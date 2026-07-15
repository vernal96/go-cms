package postgres

import "time"

type Config struct {
	Host            string        `envconfig:"POSTGRES_HOST" default:"localhost"`
	Port            int           `envconfig:"POSTGRES_PORT" default:"5432"`
	Database        string        `envconfig:"POSTGRES_DB" default:"go_cms"`
	User            string        `envconfig:"POSTGRES_USER" default:"go_cms"`
	Password        string        `envconfig:"POSTGRES_PASSWORD" default:"go_cms"`
	SSLMode         string        `envconfig:"POSTGRES_SSL_MODE" default:"disable"`
	MaxOpenConns    int           `envconfig:"POSTGRES_MAX_OPEN_CONNS" default:"10"`
	MaxIdleConns    int           `envconfig:"POSTGRES_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"POSTGRES_CONN_MAX_LIFETIME" default:"30m"`
	ConnectTimeout  time.Duration `envconfig:"POSTGRES_CONNECT_TIMEOUT" default:"5s"`
}
