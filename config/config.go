package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/ini.v1"
)

// Config holds all runtime settings.
type Config struct {
	Host     string
	Port     int
	LogLevel string // "debug", "info", "warn", "error"
	SESSDATA string // Bilibili login cookie — enables full danmaku delivery
}

// Default returns safe defaults used when no config file exists.
func Default() *Config {
	return &Config{
		Host:     "127.0.0.1",
		Port:     12450,
		LogLevel: "info",
	}
}

// defaultINI is written to disk on first run so users have something to edit.
const defaultINI = `[server]
; Listening address.
;   127.0.0.1  — only this computer can connect (default, recommended)
;   0.0.0.0    — allow other devices on your local network
host = 127.0.0.1

; Port number. Change this if 12450 is already in use on your system.
port = 12450

[log]
; Log level: debug, info, warn, error. Override with LOG_LEVEL env var.
level = info

[bilibili]
; Your Bilibili SESSDATA cookie value (from browser DevTools → Application → Cookies).
; Without this, Bilibili only delivers a fraction of danmaku in busy rooms.
; sessdata =
`

// Load reads config from path.
// If the file does not exist it is created with defaults and those defaults
// are returned. A parse error returns nil and the error.
func Load(path string) (*Config, error) {
	cfg := Default()

	// Environment variable always takes precedence.
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return cfg, fmt.Errorf("create config directory: %w", err)
		}
		if err := os.WriteFile(path, []byte(defaultINI), 0o644); err != nil {
			return cfg, fmt.Errorf("write default config: %w", err)
		}
		return cfg, nil
	}

	f, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}

	s := f.Section("server")
	cfg.Host = s.Key("host").MustString(cfg.Host)
	cfg.Port = s.Key("port").MustInt(cfg.Port)

	l := f.Section("log")
	cfg.LogLevel = l.Key("level").MustString(cfg.LogLevel)

	// Environment variable takes precedence over config file.
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	b := f.Section("bilibili")
	cfg.SESSDATA = b.Key("sessdata").MustString("")
	if v := os.Getenv("SESSDATA"); v != "" {
		cfg.SESSDATA = v
	}

	return cfg, nil
}
