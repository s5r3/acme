// Package assets embeds the web binary's static files (CSS, JS, images).
package assets

import (
	"embed"
	"io/fs"
)

//go:embed static
var files embed.FS

// FS is served under /static/ by the web binary.
var FS, _ = fs.Sub(files, "static")
