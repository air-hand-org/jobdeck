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
	defaultPostgresUser     = "postgres"
	defaultPostgresHost     = "127.0.0.1"
	defaultPostgresPort     = "5432"
	defaultPostgresDatabase = "jobdeck-auth"
	defaultPostgresSSLMode  = "disable"

	migratorPath = "/auth-migrator"
	serverPath   = "/auth"

	migratorStartupDeadline = 30 * time.Second
	migratorRetryInterval   = time.Second
	migratorWaitDelay       = 5 * time.Second
)

func environmentOrDefault(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return defaultValue
}

func postgresURIFromEnvironment() (string, error) {
	if connectionURI := os.Getenv("POSTGRESDB_URI"); connectionURI != "" {
		return connectionURI, nil
	}

	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		return "", fmt.Errorf("POSTGRES_PASSWORD is required when POSTGRESDB_URI is not set")
	}

	postgresURL := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			environmentOrDefault("POSTGRES_USER", defaultPostgresUser),
			password,
		),
		Host: net.JoinHostPort(
			environmentOrDefault("POSTGRES_HOST", defaultPostgresHost),
			environmentOrDefault("POSTGRES_PORT", defaultPostgresPort),
		),
		Path: environmentOrDefault("POSTGRES_DATABASE", defaultPostgresDatabase),
	}

	query := postgresURL.Query()
	query.Set("sslmode", environmentOrDefault("POSTGRES_SSLMODE", defaultPostgresSSLMode))
	postgresURL.RawQuery = query.Encode()

	return postgresURL.String(), nil
}

func executeMigrator(ctx context.Context) error {
	command := exec.CommandContext(ctx, migratorPath)
	// CommandContext uses SIGKILL by default. Let the migrator close its database
	// connection on a Cloud Run termination signal before enforcing WaitDelay.
	command.Cancel = func() error {
		if command.Process == nil {
			return nil
		}

		return command.Process.Signal(syscall.SIGTERM)
	}
	command.WaitDelay = migratorWaitDelay
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	return command.Run()
}

func runMigrator(ctx context.Context, startupDeadline, retryInterval time.Duration, run func(context.Context) error) error {
	migratorCtx, cancel := context.WithTimeout(ctx, startupDeadline)
	defer cancel()

	var lastErr error
	for attempt := 1; ; attempt++ {
		if err := migratorCtx.Err(); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				if lastErr == nil {
					return fmt.Errorf("migrator did not start before the %s startup deadline", startupDeadline)
				}

				return fmt.Errorf("migrator did not succeed within the %s startup deadline: %w", startupDeadline, lastErr)
			}

			return fmt.Errorf("migrator canceled: %w", err)
		}

		if err := run(migratorCtx); err == nil {
			return nil
		} else {
			lastErr = err
		}

		log.Printf("migrator attempt %d failed: %v; retrying in %s", attempt, lastErr, retryInterval)
		timer := time.NewTimer(retryInterval)
		select {
		case <-migratorCtx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}

func main() {
	signalCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	postgresURI, err := postgresURIFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	if err := os.Setenv("POSTGRESDB_URI", postgresURI); err != nil {
		log.Fatalf("set POSTGRESDB_URI: %v", err)
	}

	if err := runMigrator(signalCtx, migratorStartupDeadline, migratorRetryInterval, executeMigrator); err != nil {
		log.Fatal(err)
	}

	if err := syscall.Exec(serverPath, []string{serverPath}, os.Environ()); err != nil {
		log.Fatalf("start server: %v", err)
	}
}
