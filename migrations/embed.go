// Package migrations embeds the versioned goose SQL files so the migrate
// binary and integration tests carry them with no extra files.
package migrations

import "embed"

// FS holds every *.sql migration.
//
//go:embed *.sql
var FS embed.FS
