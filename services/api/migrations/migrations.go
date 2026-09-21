// Package migrations embute os arquivos SQL no binário.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
