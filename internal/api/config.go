package api

import (
	"encoding/json"
	"net/http"

	"github.com/iucario/bili-danmu-go/internal/config"
)

type sessdataPatchRequest struct {
	SESSDATA string `json:"sessdata"`
}

// NewConfigHandler returns a handler for the persisted backend config used by main.go.
//
//	GET /api/config   — export the current editable config file values
//	POST /api/config  — import and save the full config
//	PATCH /api/config — update only sessdata
func NewConfigHandler(store *config.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			handleGetConfig(w, store)

		case http.MethodPost:
			handlePostConfig(w, r, store)

		case http.MethodPatch:
			handlePatchConfig(w, r, store)

		default:
			w.Header().Set("Allow", "GET, POST, PATCH")
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		}
	})
}

func handleGetConfig(w http.ResponseWriter, store *config.Store) {
	cfg, err := store.LoadEditable()
	if err != nil {
		http.Error(w, jsonError(err.Error()), http.StatusInternalServerError)
		return
	}
	writeJSON(w, cfg)
}

func handlePostConfig(w http.ResponseWriter, r *http.Request, store *config.Store) {
	var cfg config.Config
	if err := decodeJSON(r, &cfg); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	if err := cfg.Validate(); err != nil {
		http.Error(w, jsonError(err.Error()), http.StatusBadRequest)
		return
	}
	stored, err := store.Save(cfg)
	if err != nil {
		http.Error(w, jsonError(err.Error()), http.StatusInternalServerError)
		return
	}
	writeJSON(w, stored)
}

func handlePatchConfig(w http.ResponseWriter, r *http.Request, store *config.Store) {
	var patch sessdataPatchRequest
	if err := decodeJSON(r, &patch); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	cfg, err := store.LoadEditable()
	if err != nil {
		http.Error(w, jsonError(err.Error()), http.StatusInternalServerError)
		return
	}
	cfg.SESSDATA = patch.SESSDATA

	stored, err := store.Save(*cfg)
	if err != nil {
		http.Error(w, jsonError(err.Error()), http.StatusInternalServerError)
		return
	}
	writeJSON(w, stored)
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, v any) {
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(msg string) string {
	b, _ := json.Marshal(map[string]string{"error": msg})
	return string(b)
}
