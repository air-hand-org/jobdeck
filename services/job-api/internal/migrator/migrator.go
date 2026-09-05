// Package migrator applies and reverts the job API database migrations.
package migrator

import (
	"context"
	"database/sql"

	migrations "jobdeck/services/job-api/db/job"

	_ "github.com/jackc/pgx/v5/stdlib" // Register the pgx database/sql driver.
	"github.com/pressly/goose/v3"
)

const migrationsDir = "migrations"

func configure() error {
	goose.SetBaseFS(migrations.FS)
	goose.SetTableName("public.goose_db_version")
	return goose.SetDialect("postgres")
}

// Up applies all pending database migrations.
func Up(ctx context.Context, connectionString string) error {
	return withDB(connectionString, func(db *sql.DB) error {
		return up(ctx, db)
	})
}

// DownTo0 reverts all database migrations.
func DownTo0(ctx context.Context, connectionString string) error {
	return withDB(connectionString, func(db *sql.DB) error {
		return downTo0(ctx, db)
	})
}

func withDB(connectionString string, fn func(*sql.DB) error) error {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	return fn(db)
}

func up(ctx context.Context, db *sql.DB) error {
	if err := configure(); err != nil {
		return err
	}
	return goose.UpContext(ctx, db, migrationsDir)
}

func downTo0(ctx context.Context, db *sql.DB) error {
	if err := configure(); err != nil {
		return err
	}
	return goose.DownToContext(ctx, db, migrationsDir, 0)
}
