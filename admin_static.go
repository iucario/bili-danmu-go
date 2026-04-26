//go:build obs

package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:apps/admin-web/build
var adminWebFS embed.FS

// adminHandler returns an http.Handler that serves the admin UI static site
// under the /admin/ path prefix.
func adminHandler() http.Handler {
	sub, err := fs.Sub(adminWebFS, "apps/admin-web/build")
	if err != nil {
		panic("admin embed sub: " + err.Error())
	}
	return http.StripPrefix("/admin", http.FileServer(http.FS(sub)))
}
