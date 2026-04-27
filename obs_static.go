//go:build obs

package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:apps/obs-plugin/build
var obsPluginFS embed.FS

// obsHandler returns an http.Handler that serves the OBS overlay static site
// under the /obs/ path prefix.
func obsHandler() http.Handler {
	// Strip the leading "apps/obs-plugin/build" from embedded paths so the
	// handler sees the files rooted at the build output directory.
	sub, err := fs.Sub(obsPluginFS, "apps/obs-plugin/build")
	if err != nil {
		panic("obs embed sub: " + err.Error())
	}
	return http.StripPrefix("/obs", http.FileServer(http.FS(sub)))
}

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
