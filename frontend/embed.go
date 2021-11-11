// Package frontend embeds static frontend resources
package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed dist
var dist embed.FS

// StaticHandler handles static requests
var StaticHandler http.Handler

func init() {
	dist, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	StaticHandler = http.FileServer(http.FS(dist))
}
