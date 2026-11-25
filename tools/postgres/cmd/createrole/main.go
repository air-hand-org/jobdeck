package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func withDB(ctx context.Context, connString string, fn func(context.Context, *sql.DB) error) error {
	cfg, err := pgx.ParseConfig(connString)
	if err != nil {
		return err
	}
	db := stdlib.OpenDB(*cfg)
	defer func() { _ = db.Close() }()

	return fn(ctx, db)
}

type config struct {
	service string
}

func (c *config) Validate() error {
	if len(c.service) == 0 {
		return fmt.Errorf("service must not be empty")
	}
	return nil
}

func parseFlags() config {
	var cfg config

	flag.StringVar(&cfg.service, "service", "", "service name")
	flag.Parse()

	return cfg
}

func mainLogic(ctx context.Context, logger *slog.Logger) error {
	postgreConnString, found := os.LookupEnv("POSTGRES_URI")
	if !found {
		return fmt.Errorf("POSTGRES_URL is required")
	}
	cfg := parseFlags()
	if err := cfg.Validate(); err != nil {
		return err
	}
	return withDB(ctx, postgreConnString, func(ctx context.Context, db *sql.DB) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()

		if _, err := tx.ExecContext(ctx, "SELECT create_migrator_role($1)", cfg.service); err != nil {
			return fmt.Errorf("create role: %s %w", cfg.service, err)
		}
		if _, err := tx.ExecContext(ctx, "SELECT create_runtime_role($1)", cfg.service); err != nil {
			return fmt.Errorf("create role: %s %w", cfg.service, err)
		}
		return tx.Commit()
	})
}

func main() {
	signalCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := slog.Default()
	if err := mainLogic(signalCtx, logger); err != nil {
		logger.Error("error occured", "error", err)
		os.Exit(1)
	}
}
