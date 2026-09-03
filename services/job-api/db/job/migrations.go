// Package migrations embeds the job database migrations.
package migrations

import (
	"embed"
)

// FS contains the embedded job database migrations.
//
//go:embed migrations/*.sql
var FS embed.FS
