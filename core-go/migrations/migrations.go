// Package migrations provides the embedded SQL migration files.
package migrations

import "embed"

// FS contains all migration SQL files for golang-migrate iofs source.
//go:embed *.sql
var FS embed.FS
