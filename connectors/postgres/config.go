package postgres

type Config struct {
	Host string `envconfig:"POSTGRES_HOST" default:"localhost"`
	Port int    `envconfig:"POSTGRES_PORT" default:"5432"`
}
