package mysqldb

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

//go:embed schema.sql
var schema string

type Database struct {
	db *sql.DB
}

func Connect(ctx context.Context, dsn string) (*Database, error) {
	if dsn == "" {
		return nil, errors.New("mysql DSN is empty")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return &Database{
		db: db,
	}, nil
}

func (d *Database) DB() *sql.DB {
	return d.db
}

func (d *Database) Migrate(ctx context.Context) error {
	return Migrate(ctx, d.db)
}

func (d *Database) Close() error {
	return d.db.Close()
}

func Migrate(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("mysql migration db is nil")
	}

	if _, err := db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("migrate mysql schema: %w", err)
	}

	return nil
}
