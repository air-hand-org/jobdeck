package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"

	"jobdeck/services/auth/internal/migrator"

	_ "github.com/jackc/pgx/v5/stdlib"
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

	ctx := context.Background()
	if cfg.downTo0 {
		err = migrator.DownTo0(ctx, db)
	} else {
		err = migrator.Up(ctx, db)
	}
	if err != nil {
		log.Fatal(err)
	}
}
