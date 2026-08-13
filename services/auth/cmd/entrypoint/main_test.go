package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresURIFromEnvironment(t *testing.T) {
	t.Setenv("POSTGRESDB_URI", "postgres://existing-uri")
	t.Setenv("POSTGRES_PASSWORD", "ignored")

	uri, err := postgresURIFromEnvironment()
	require.NoError(t, err)
	assert.Equal(t, "postgres://existing-uri", uri)
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
	require.NoError(t, err)
	const want = "postgres://user%3Aname:p%40ss%3A%2F%3F%20word@[::1]:6543/db/name?sslmode=verify-full"
	assert.Equal(t, want, uri)
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
	require.NoError(t, err)
	const want = "postgres://postgres:password@127.0.0.1:5432/jobdeck-auth?sslmode=disable"
	assert.Equal(t, want, uri)
}

func TestPostgresURIFromEnvironmentRequiresPassword(t *testing.T) {
	t.Setenv("POSTGRESDB_URI", "")
	t.Setenv("POSTGRES_PASSWORD", "")

	_, err := postgresURIFromEnvironment()
	require.Error(t, err)
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
	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestRunMigratorStopsAtDeadline(t *testing.T) {
	err := runMigrator(context.Background(), 10*time.Millisecond, 0, func(context.Context) error {
		return errors.New("database is not ready")
	})
	require.Error(t, err)
}

func TestRunMigratorStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := runMigrator(ctx, time.Second, 0, func(context.Context) error {
		require.FailNow(t, "migrator must not run after cancellation")
		return nil
	})
	require.Error(t, err)
}
