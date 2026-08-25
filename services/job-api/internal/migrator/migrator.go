package migrator

import (
	"context"
	"database/sql"

	migrations "jobdeck/services/job-api/db/job"

	"github.com/pressly/goose/v3"
)

const migrationsDir = "migrations"

func configure() error {
	goose.SetBaseFS(migrations.FS)
	goose.SetTableName("public.goose_db_version")
	return goose.SetDialect("postgres")
}

func Up(ctx context.Context, db *sql.DB) error {
	if err := configure(); err != nil {
		return err
	}
	return goose.UpContext(ctx, db, migrationsDir)
}

func DownTo0(ctx context.Context, db *sql.DB) error {
	if err := configure(); err != nil {
		return err
	}
	return goose.DownToContext(ctx, db, migrationsDir, 0)
}
