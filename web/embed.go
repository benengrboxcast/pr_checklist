// Package web embeds the static frontend assets into the compiled binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var embedded embed.FS

// StaticFS is the embedded frontend, rooted so "index.html" etc. are at the top level.
var StaticFS = mustSub(embedded, "static")

func mustSub(f embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
