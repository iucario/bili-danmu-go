//go:build !obs

package main

import (
	"net/http"
	"os"
)

// obsHandler serves the OBS overlay from the filesystem at runtime so that
// changes from `pnpm build` are visible immediately without restarting Go.
// Build with -tags obs to embed the files into a self-contained binary instead.
func obsHandler() http.Handler {
	const buildDir = "apps/obs-plugin/build"
	return http.StripPrefix("/obs", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(buildDir); os.IsNotExist(err) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("OBS overlay not built.\n\nRun: cd apps/obs-plugin && pnpm install && pnpm build\n"))
			return
		}
		http.FileServer(http.Dir(buildDir)).ServeHTTP(w, r)
	}))
}
