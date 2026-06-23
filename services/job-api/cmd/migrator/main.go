package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	migrations "jobdeck/services/job-api/db/job"
)

func main() {
	connStr := os.Getenv("POSTGRESDB_URI")
	if connStr == "" {
		log.Fatal("POSTGRESDB_URI is required")
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	goose.SetBaseFS(migrations.FS)
	goose.SetTableName("public.goose_db_version")

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatal(err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		log.Fatal(err)
	}
}
