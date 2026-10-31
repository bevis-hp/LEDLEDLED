// Package webui embeds the built Svelte frontend (frontend/ -> dist/) so the
// whole app ships as a single binary.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// FS returns the built frontend files rooted at dist/.
func FS() fs.FS {
	sub, _ := fs.Sub(dist, "dist")
	return sub
}
