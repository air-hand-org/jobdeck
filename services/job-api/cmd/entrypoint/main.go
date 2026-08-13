package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

const (
	migratorPath = "/job-api-migrator"
	serverPath   = "/job-api"

	migratorAttempts      = 30
	migratorRetryInterval = time.Second
)

func envOrDefault(getenv func(string) string, key, defaultValue string) string {
	if value := getenv(key); value != "" {
		return value
	}

	return defaultValue
}

// postgresURI returns the existing explicit URI when provided. Cloud Run injects
// POSTGRES_PASSWORD from Secret Manager at runtime, so the fallback keeps the
// password out of the deployment configuration and OpenTofu state.
func postgresURI(getenv func(string) string) (string, error) {
	if value := getenv("POSTGRESDB_URI"); value != "" {
		return value, nil
	}

	password := getenv("POSTGRES_PASSWORD")
	if password == "" {
		return "", errors.New("POSTGRESDB_URI is required when POSTGRES_PASSWORD is not set")
	}

	query := url.Values{}
	query.Set("sslmode", envOrDefault(getenv, "POSTGRES_SSLMODE", "disable"))

	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(envOrDefault(getenv, "POSTGRES_USER", "postgres"), password),
		Host:     net.JoinHostPort(envOrDefault(getenv, "POSTGRES_HOST", "127.0.0.1"), envOrDefault(getenv, "POSTGRES_PORT", "5432")),
		Path:     "/" + envOrDefault(getenv, "POSTGRES_DATABASE", "jobdeck"),
		RawQuery: query.Encode(),
	}).String(), nil
}

func runMigrator(ctx context.Context) error {
	command := exec.CommandContext(ctx, migratorPath)
	command.Env = os.Environ()
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func runMigratorUntilReady(ctx context.Context) error {
	var lastErr error
	for attempt := 1; attempt <= migratorAttempts; attempt++ {
		if err := runMigrator(ctx); err == nil {
			return nil
		} else if ctx.Err() != nil {
			return ctx.Err()
		} else {
			lastErr = err
		}

		if attempt == migratorAttempts {
			break
		}

		log.Printf("migrator attempt %d/%d failed: %v; retrying in %s", attempt, migratorAttempts, lastErr, migratorRetryInterval)

		timer := time.NewTimer(migratorRetryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	return fmt.Errorf("migrator failed after %d attempts: %w", migratorAttempts, lastErr)
}

func main() {
	uri, err := postgresURI(os.Getenv)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.Setenv("POSTGRESDB_URI", uri); err != nil {
		log.Fatalf("set POSTGRESDB_URI: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runMigratorUntilReady(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		log.Fatalf("run migrator: %v", err)
	}

	if err := syscall.Exec(serverPath, []string{serverPath}, os.Environ()); err != nil {
		log.Fatal(fmt.Errorf("exec server: %w", err))
	}
}
