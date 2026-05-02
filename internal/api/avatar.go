package api

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var avatarClient = &http.Client{Timeout: 5 * time.Second}

// NewAvatarProxyHandler returns a handler for GET /api/avatar?url=<bilibili-face-url>.
// It fetches the image server-side (bypassing browser CORS) and streams it back.
func NewAvatarProxyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.URL.Query().Get("url")
		if raw == "" {
			http.Error(w, "missing url", http.StatusBadRequest)
			return
		}

		parsed, err := url.Parse(raw)
		if err != nil || !strings.HasSuffix(parsed.Host, ".hdslb.com") {
			http.Error(w, "invalid url", http.StatusBadRequest)
			return
		}

		resp, err := avatarClient.Get(raw)
		if err != nil || resp.StatusCode != http.StatusOK {
			http.Error(w, "upstream error", http.StatusBadGateway)
			return
		}
		defer func() { _ = resp.Body.Close() }()

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.Header().Set("Cache-Control", "public, max-age=86400")
		if _, err := io.Copy(w, resp.Body); err != nil {
			// Response headers already sent; nothing useful we can do.
			return
		}
	})
}
