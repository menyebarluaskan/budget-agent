package web

import "embed"

//go:embed all:static
var Build embed.FS
