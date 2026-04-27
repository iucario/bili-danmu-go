package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

// Config holds all runtime settings.
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	LogLevel string `json:"logLevel"` // "debug", "info", "warn", "error"
	SESSDATA string `json:"sessdata"` // Bilibili login cookie — enables full danmaku delivery
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

var validLogLevels = map[string]struct{}{
	"debug": {},
	"info":  {},
	"warn":  {},
	"error": {},
}

// Load reads config from path.
// If the file does not exist it is created with defaults and those defaults
// are returned. A parse error returns nil and the error.
func Load(path string) (*Config, error) {
	return load(path, true)
}

// LoadEditable reads config values from disk without applying environment overrides.
func LoadEditable(path string) (*Config, error) {
	return load(path, false)
}

func load(path string, applyEnv bool) (*Config, error) {
	cfg := Default()

	if err := ensureConfigFile(path); err != nil {
		return cfg, err
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

	b := f.Section("bilibili")
	cfg.SESSDATA = b.Key("sessdata").MustString("")

	if applyEnv {
		applyEnvOverrides(cfg)
	}

	return cfg, nil
}

// Save writes the config to path using the same INI shape as the default config file.
func (cfg *Config) Save(path string) error {
	normalized := cfg.normalized()
	if err := normalized.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	f := ini.Empty()
	f.Section("server").Key("host").SetValue(normalized.Host)
	f.Section("server").Key("port").SetValue(fmt.Sprintf("%d", normalized.Port))
	f.Section("log").Key("level").SetValue(normalized.LogLevel)
	f.Section("bilibili").Key("sessdata").SetValue(normalized.SESSDATA)

	if err := f.SaveTo(path); err != nil {
		return fmt.Errorf("write config %q: %w", path, err)
	}
	return nil
}

// Validate checks whether the config contains supported values.
func (cfg *Config) Validate() error {
	normalized := cfg.normalized()
	if normalized.Host == "" {
		return fmt.Errorf("host is required")
	}
	if normalized.Port < 1 || normalized.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if _, ok := validLogLevels[normalized.LogLevel]; !ok {
		return fmt.Errorf("log level must be one of debug, info, warn, error")
	}
	return nil
}

// ActiveEnvOverrideFields lists config fields currently overridden by environment variables.
func ActiveEnvOverrideFields() []string {
	fields := make([]string, 0, 2)
	if os.Getenv("LOG_LEVEL") != "" {
		fields = append(fields, "logLevel")
	}
	if os.Getenv("SESSDATA") != "" {
		fields = append(fields, "sessdata")
	}
	return fields
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("SESSDATA"); v != "" {
		cfg.SESSDATA = v
	}
}

func ensureConfigFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create config directory: %w", err)
		}
		if err := os.WriteFile(path, []byte(defaultINI), 0o644); err != nil {
			return fmt.Errorf("write default config: %w", err)
		}
	}
	return nil
}

func (cfg *Config) normalized() Config {
	copy := *cfg
	copy.Host = strings.TrimSpace(copy.Host)
	copy.LogLevel = strings.ToLower(strings.TrimSpace(copy.LogLevel))
	copy.SESSDATA = strings.TrimSpace(copy.SESSDATA)
	return copy
}
