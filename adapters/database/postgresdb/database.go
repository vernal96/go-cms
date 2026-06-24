package postgresdb

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

type Database struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, dsn string) (*Database, error) {
	if dsn == "" {
		return nil, errors.New("postgres DSN is empty")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Database{
		pool: pool,
	}, nil
}

func (d *Database) Pool() *pgxpool.Pool {
	return d.pool
}

func (d *Database) Migrate(ctx context.Context) error {
	return Migrate(ctx, d.pool)
}

func (d *Database) Close() {
	d.pool.Close()
}

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return errors.New("postgres migration pool is nil")
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("migrate postgres schema: %w", err)
	}

	return nil
}
