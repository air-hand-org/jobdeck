// Command migrator applies database migrations.
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"jobdeck/services/job-api/internal/migrator"
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

	ctx := context.Background()
	var err error
	if cfg.downTo0 {
		err = migrator.DownTo0(ctx, connStr)
	} else {
		err = migrator.Up(ctx, connStr)
	}
	if err != nil {
		log.Fatal(err)
	}
}
