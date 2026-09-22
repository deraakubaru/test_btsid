package migrations

import "embed"

// MigrationsFS embeds all SQL migration files into the compiled binary.
//
//go:embed *.sql
var MigrationsFS embed.FS
