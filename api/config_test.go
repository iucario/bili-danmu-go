package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	appconfig "github.com/iucario/bili-danmu-go/config"
	runtimeconfig "github.com/iucario/bili-danmu-go/internal/appconfig"
)

func TestConfigHandlerReadsAndWritesConfigFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.ini")
	current, err := appconfig.Load(configPath)
	if err != nil {
		t.Fatalf("load initial config: %v", err)
	}
	store := runtimeconfig.New(configPath, current, func(prev, next *appconfig.Config) error {
		return nil
	})
	handler := NewConfigHandler(store)

	getReq := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	getResp := httptest.NewRecorder()
	handler.ServeHTTP(getResp, getReq)

	if getResp.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d", getResp.Code, http.StatusOK)
	}

	var initial appconfig.Config
	if err := json.NewDecoder(getResp.Body).Decode(&initial); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if initial.Host != "127.0.0.1" || initial.Port != 12450 || initial.LogLevel != "info" {
		t.Fatalf("unexpected default config: %+v", initial)
	}

	body, err := json.Marshal(appconfig.Config{
		Host:     "0.0.0.0",
		Port:     18080,
		LogLevel: "warn",
		SESSDATA: "cookie-value",
	})
	if err != nil {
		t.Fatalf("marshal POST body: %v", err)
	}

	postReq := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	postResp := httptest.NewRecorder()
	handler.ServeHTTP(postResp, postReq)

	if postResp.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want %d body=%s", postResp.Code, http.StatusOK, postResp.Body.String())
	}

	var storedCfg appconfig.Config
	if err := json.NewDecoder(postResp.Body).Decode(&storedCfg); err != nil {
		t.Fatalf("decode POST response: %v", err)
	}
	if storedCfg.Host != "0.0.0.0" || storedCfg.Port != 18080 || storedCfg.LogLevel != "warn" || storedCfg.SESSDATA != "cookie-value" {
		t.Fatalf("unexpected POST response config: %+v", storedCfg)
	}

	patchReq := httptest.NewRequest(http.MethodPatch, "/api/config", bytes.NewReader([]byte(`{"sessdata":"patched-cookie"}`)))
	patchResp := httptest.NewRecorder()
	handler.ServeHTTP(patchResp, patchReq)

	if patchResp.Code != http.StatusOK {
		t.Fatalf("PATCH status = %d, want %d body=%s", patchResp.Code, http.StatusOK, patchResp.Body.String())
	}

	var patchedCfg appconfig.Config
	if err := json.NewDecoder(patchResp.Body).Decode(&patchedCfg); err != nil {
		t.Fatalf("decode PATCH response: %v", err)
	}
	if patchedCfg.SESSDATA != "patched-cookie" {
		t.Fatalf("unexpected patched sessdata: %+v", patchedCfg)
	}

	stored, err := appconfig.LoadEditable(configPath)
	if err != nil {
		t.Fatalf("load saved config: %v", err)
	}
	if stored.Host != "0.0.0.0" || stored.Port != 18080 || stored.LogLevel != "warn" || stored.SESSDATA != "patched-cookie" {
		t.Fatalf("unexpected stored config: %+v", stored)
	}
}

func TestConfigHandlerRejectsInvalidConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.ini")
	current, err := appconfig.Load(configPath)
	if err != nil {
		t.Fatalf("load initial config: %v", err)
	}
	store := runtimeconfig.New(configPath, current, nil)
	handler := NewConfigHandler(store)

	body := []byte(`{"host":"127.0.0.1","port":70000,"logLevel":"info","sessdata":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/config", bytes.NewReader(body))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("POST status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
}

func TestConfigHandlerRejectsInvalidPatch(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.ini")
	current, err := appconfig.Load(configPath)
	if err != nil {
		t.Fatalf("load initial config: %v", err)
	}
	store := runtimeconfig.New(configPath, current, nil)
	handler := NewConfigHandler(store)

	req := httptest.NewRequest(http.MethodPatch, "/api/config", bytes.NewReader([]byte(`{"host":"127.0.0.1"}`)))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("PATCH status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
}
