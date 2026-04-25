package api

import (
	"encoding/json"
	"net/http"

	"github.com/iucario/bili-danmu-go/internal/appconfig"
)

// NewConfigHandler returns a handler for /api/config.
//
//	GET  /api/config  — return current config as JSON
//	PUT  /api/config  — replace entire config (validates all fields)
//	POST /api/config  — partial update, merges non-zero fields
func NewConfigHandler(store *appconfig.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			writeJSON(w, store.Get())

		case http.MethodPut:
			var cfg appconfig.AppConfig
			if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
				http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
				return
			}
			if err := store.Set(cfg); err != nil {
				http.Error(w, jsonError(err.Error()), http.StatusBadRequest)
				return
			}
			writeJSON(w, store.Get())

		case http.MethodPost:
			var patch appconfig.AppConfig
			if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
				http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
				return
			}
			if err := store.Patch(patch); err != nil {
				http.Error(w, jsonError(err.Error()), http.StatusBadRequest)
				return
			}
			writeJSON(w, store.Get())

		default:
			w.Header().Set("Allow", "GET, PUT, POST")
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}
