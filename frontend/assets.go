package frontend

import "embed"

// Assets contains the production frontend bundled by Vite.
//
//go:embed all:dist
var Assets embed.FS
