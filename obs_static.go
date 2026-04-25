//go:build obs

package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed apps/obs-plugin/build
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
