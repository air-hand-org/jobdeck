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
	sqlfile string
}

func parseFlags() config {
	var cfg config

	flag.StringVar(&cfg.sqlfile, "sqlfile", "", "sql filepath")
	flag.Parse()

	return cfg
}

func mainLogic(ctx context.Context, logger *slog.Logger) error {
	postgreConnString, found := os.LookupEnv("POSTGRES_URI")
	if !found {
		return fmt.Errorf("POSTGRES_URL is required")
	}
	cfg := parseFlags()
	sqlcontent, err := os.ReadFile(cfg.sqlfile)
	if err != nil {
		return err
	}

	if err := withDB(ctx, postgreConnString, func(ctx context.Context, db *sql.DB) error {
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		defer func() { _ = tx.Rollback() }()

		if _, err := tx.ExecContext(ctx, string(sqlcontent)); err != nil {
			return fmt.Errorf("execute %s: %w", cfg.sqlfile, err)
		}
		return tx.Commit()
	}); err != nil {
		return err
	}
	logger.InfoContext(ctx, "sql executed", "file", cfg.sqlfile)
	return nil
}

func main() {
	signalCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := slog.Default()
	if err := mainLogic(signalCtx, logger); err != nil {
		logger.ErrorContext(signalCtx, "error occured", "error", err)
		os.Exit(1)
	}
}
