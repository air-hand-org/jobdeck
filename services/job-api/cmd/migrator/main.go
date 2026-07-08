package main

import (
	"database/sql"
	"flag"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	migrations "jobdeck/services/job-api/db/job"
)

type config struct {
	downTo0 bool
}

func parseFlags() config {
	var cfg config

	flag.BoolVar(&cfg.downTo0, "down-to-0", false, "down to 0")
	flag.Parse()

	return cfg
}

func main() {
	cfg := parseFlags()

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

	if cfg.downTo0 {
		if err := goose.DownTo(db, "migrations", 0); err != nil {
			log.Fatal(err)
		}
	} else {
		if err := goose.Up(db, "migrations"); err != nil {
			log.Fatal(err)
		}
	}
}
