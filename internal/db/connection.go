package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	defaultMaxOpenConns = 10
	defaultMaxIdleConns = 5
	defaultConnMaxAge   = 5 * time.Minute
)

type Connection struct {
	DSN string
	DB  *sql.DB
}

func NewConnection(dsn string) (*Connection, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("db dsn is required")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetConnMaxLifetime(defaultConnMaxAge)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Connection{DSN: dsn, DB: db}, nil
}

func (c *Connection) Ping(ctx context.Context) error {
	if c == nil || c.DB == nil {
		return errors.New("db connection not initialized")
	}
	return c.DB.PingContext(ctx)
}

func (c *Connection) Close() error {
	if c == nil || c.DB == nil {
		return nil
	}
	return c.DB.Close()
}
