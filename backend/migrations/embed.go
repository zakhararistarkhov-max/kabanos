// Package migrations embeds the SQL migration files so they ship inside the
// single API binary. Migrations are applied on startup (guarded by a Postgres
// advisory lock) which keeps deployments — including rolling multi-replica
// deployments — simple and reproducible.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
