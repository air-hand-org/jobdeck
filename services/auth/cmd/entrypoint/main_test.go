package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPostgresURIFromEnvironment(t *testing.T) {
	t.Setenv("POSTGRESDB_URI", "postgres://existing-uri")
	t.Setenv("POSTGRES_PASSWORD", "ignored")

	uri, err := postgresURIFromEnvironment()
	if err != nil {
		t.Fatalf("postgresURIFromEnvironment() error = %v", err)
	}
	if want := "postgres://existing-uri"; uri != want {
		t.Errorf("postgresURIFromEnvironment() = %q, want %q", uri, want)
	}
}

func TestPostgresURIFromEnvironmentBuildsEscapedURI(t *testing.T) {
	t.Setenv("POSTGRESDB_URI", "")
	t.Setenv("POSTGRES_PASSWORD", "p@ss:/? word")
	t.Setenv("POSTGRES_USER", "user:name")
	t.Setenv("POSTGRES_HOST", "::1")
	t.Setenv("POSTGRES_PORT", "6543")
	t.Setenv("POSTGRES_DATABASE", "db/name")
	t.Setenv("POSTGRES_SSLMODE", "verify-full")

	uri, err := postgresURIFromEnvironment()
	if err != nil {
		t.Fatalf("postgresURIFromEnvironment() error = %v", err)
	}
	const want = "postgres://user%3Aname:p%40ss%3A%2F%3F%20word@[::1]:6543/db/name?sslmode=verify-full"
	if uri != want {
		t.Errorf("postgresURIFromEnvironment() = %q, want %q", uri, want)
	}
}

func TestPostgresURIFromEnvironmentUsesDefaults(t *testing.T) {
	t.Setenv("POSTGRESDB_URI", "")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_USER", "")
	t.Setenv("POSTGRES_HOST", "")
	t.Setenv("POSTGRES_PORT", "")
	t.Setenv("POSTGRES_DATABASE", "")
	t.Setenv("POSTGRES_SSLMODE", "")

	uri, err := postgresURIFromEnvironment()
	if err != nil {
		t.Fatalf("postgresURIFromEnvironment() error = %v", err)
	}
	const want = "postgres://postgres:password@127.0.0.1:5432/jobdeck-auth?sslmode=disable"
	if uri != want {
		t.Errorf("postgresURIFromEnvironment() = %q, want %q", uri, want)
	}
}

func TestPostgresURIFromEnvironmentRequiresPassword(t *testing.T) {
	t.Setenv("POSTGRESDB_URI", "")
	t.Setenv("POSTGRES_PASSWORD", "")

	_, err := postgresURIFromEnvironment()
	if err == nil {
		t.Fatal("postgresURIFromEnvironment() error = nil, want error")
	}
}

func TestRunMigratorRetriesUntilSuccess(t *testing.T) {
	attempts := 0
	err := runMigrator(context.Background(), time.Second, 0, func(context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("database is not ready")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("runMigrator() error = %v", err)
	}
	if attempts != 3 {
		t.Errorf("migrator attempts = %d, want 3", attempts)
	}
}

func TestRunMigratorStopsAtDeadline(t *testing.T) {
	err := runMigrator(context.Background(), 10*time.Millisecond, 0, func(context.Context) error {
		return errors.New("database is not ready")
	})
	if err == nil {
		t.Fatal("runMigrator() error = nil, want deadline error")
	}
}

func TestRunMigratorStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runMigrator(ctx, time.Second, 0, func(context.Context) error {
		t.Fatal("migrator must not run after cancellation")
		return nil
	})
	if err == nil {
		t.Fatal("runMigrator() error = nil, want cancellation error")
	}
}
