package migrations

import (
	"embed"

	// Enable go migrations
	_ "jobdeck/services/auth/internal/db/auth/migrations"
)

//go:embed migrations/*.sql
var FS embed.FS
