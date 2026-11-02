// Package migrations embeds the goose SQL migrations so trailmix-migrate
// carries them in its binary.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
