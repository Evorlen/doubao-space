package web

import (
	"embed"
	"io/fs"
)

// Include files whose names begin with "." or "_" as well; Vite emits a
// Vue helper asset with a leading underscore.
//
//go:embed all:dist
var embeddedDist embed.FS

// DistFS is the frontend build output served by the Go application.
var DistFS fs.FS

func init() {
	var err error
	DistFS, err = fs.Sub(embeddedDist, "dist")
	if err != nil {
		panic(err)
	}
}
