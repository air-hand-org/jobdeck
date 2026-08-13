package main

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresURIUsesExplicitURI(t *testing.T) {
	t.Parallel()

	got, err := postgresURI(func(key string) string {
		if key == "POSTGRESDB_URI" {
			return "postgres://existing-uri"
		}
		return ""
	})

	require.NoError(t, err)
	assert.Equal(t, "postgres://existing-uri", got)
}

func TestPostgresURIBuildsURIFromRuntimeConfiguration(t *testing.T) {
	t.Parallel()

	env := map[string]string{
		"POSTGRES_PASSWORD": "p@ss:/?word",
		"POSTGRES_USER":     "jobdeck user",
		"POSTGRES_HOST":     "postgres",
		"POSTGRES_PORT":     "5433",
		"POSTGRES_DATABASE": "jobdeck_test",
		"POSTGRES_SSLMODE":  "require",
	}
	got, err := postgresURI(func(key string) string { return env[key] })

	require.NoError(t, err)
	parsed, err := url.Parse(got)
	require.NoError(t, err)
	assert.Equal(t, "postgres", parsed.Scheme)
	assert.Equal(t, "jobdeck user", parsed.User.Username())
	password, ok := parsed.User.Password()
	require.True(t, ok)
	assert.Equal(t, "p@ss:/?word", password)
	assert.Equal(t, "postgres:5433", parsed.Host)
	assert.Equal(t, "/jobdeck_test", parsed.Path)
	assert.Equal(t, "require", parsed.Query().Get("sslmode"))
}

func TestPostgresURIUsesDefaults(t *testing.T) {
	t.Parallel()

	got, err := postgresURI(func(key string) string {
		if key == "POSTGRES_PASSWORD" {
			return "secret"
		}
		return ""
	})

	require.NoError(t, err)
	assert.Equal(t, "postgres://postgres:secret@127.0.0.1:5432/jobdeck?sslmode=disable", got)
}

func TestPostgresURIFailsWithoutURIOrPassword(t *testing.T) {
	t.Parallel()

	_, err := postgresURI(func(string) string { return "" })

	require.EqualError(t, err, "POSTGRESDB_URI is required when POSTGRES_PASSWORD is not set")
}
