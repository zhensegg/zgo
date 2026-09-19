package db

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	pool *sql.DB
}

func Open(driver, dsn string) (*DB, error) {
	pool, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("zgo/db: open %s: %w", driver, err)
	}
	if err := pool.Ping(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("zgo/db: %s: %w", driver, err)
	}
	return &DB{pool: pool}, nil
}

func OpenPostgres(dsn string) (*DB, error) {
	return Open("pgx", dsn)
}

func (d *DB) Ping(ctx context.Context) error {
	return d.pool.PingContext(ctx)
}

func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return d.pool.ExecContext(ctx, query, args...)
}

func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return d.pool.QueryContext(ctx, query, args...)
}

func (d *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return d.pool.QueryRowContext(ctx, query, args...)
}

func (d *DB) StdDB() *sql.DB { return d.pool }

func (d *DB) Close() error { return d.pool.Close() }
