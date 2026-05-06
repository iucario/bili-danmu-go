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
	TTS      TTSConfig `json:"tts"`
}

// TTSConfig holds Windows SAPI5 text-to-speech settings.
type TTSConfig struct {
	Enabled       bool   `json:"enabled"`
	RoomID        int64  `json:"roomId"`
	VoiceID       string `json:"voiceId"`       // SAPI5 voice ID substring, e.g. "ZH-CN"
	Rate          int    `json:"rate"`           // [-10, 10]
	Volume        int    `json:"volume"`         // [0, 100]
	MaxQueue      int    `json:"maxQueue"`       // max normal-priority items in queue
	MaxAgeSeconds int    `json:"maxAgeSeconds"`  // skip items older than this at speak time (0 = disabled)

	TemplateText      string `json:"templateText"`
	TemplateFreeGift  string `json:"templateFreeGift"`
	TemplatePaidGift  string `json:"templatePaidGift"`
	TemplateMember    string `json:"templateMember"`
	TemplateSuperChat string `json:"templateSuperChat"`
}

// Default returns safe defaults used when no config file exists.
func Default() *Config {
	return &Config{
		Host:     "127.0.0.1",
		Port:     5090,
		LogLevel: "info",
		TTS: TTSConfig{
			VoiceID:           "ZH-CN",
			Rate:              -2,
			Volume:            100,
			MaxQueue:          20,
			MaxAgeSeconds:     0,
			TemplateText:      "{author_name} 说，{content}",
			TemplateFreeGift:  "{author_name} 赠送了{num}个{gift_name}",
			TemplatePaidGift:  "{author_name} 赠送了{num}个{gift_name}，总价{price}元",
			TemplateMember:    "{author_name} 购买了{guard_name}",
			TemplateSuperChat: "{author_name} 发送了{price}元的醒目留言，{content}",
		},
	}
}

// defaultINI is written to disk on first run so users have something to edit.
const defaultINI = `[server]
; Listening address.
;   127.0.0.1  — only this computer can connect (default, recommended)
;   0.0.0.0    — allow other devices on your local network
host = 127.0.0.1

; Port number. Change this if 5090 is already in use on your system.
port = 5090

[log]
; Log level: debug, info, warn, error. Override with LOG_LEVEL env var.
level = info

[bilibili]
; Your Bilibili SESSDATA cookie value (from browser DevTools → Application → Cookies).
; Without this, Bilibili only delivers a fraction of danmaku in busy rooms.
; sessdata =

[tts]
; Windows SAPI5 text-to-speech. Restart is not required — changes hot-reload.
; enabled = false
; room_id = 0
; voice_id = ZH-CN
; rate = -2
; volume = 100
; max_queue = 20
; max_age_seconds = 0

; Speech templates — {placeholders} are substituted at runtime.
; template_text = {author_name} 说，{content}
; template_free_gift = {author_name} 赠送了{num}个{gift_name}
; template_paid_gift = {author_name} 赠送了{num}个{gift_name}，总价{price}元
; template_member = {author_name} 购买了{guard_name}
; template_super_chat = {author_name} 发送了{price}元的醒目留言，{content}
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

	t := f.Section("tts")
	cfg.TTS.Enabled = t.Key("enabled").MustBool(cfg.TTS.Enabled)
	cfg.TTS.RoomID = t.Key("room_id").MustInt64(cfg.TTS.RoomID)
	cfg.TTS.VoiceID = t.Key("voice_id").MustString(cfg.TTS.VoiceID)
	cfg.TTS.Rate = t.Key("rate").MustInt(cfg.TTS.Rate)
	cfg.TTS.Volume = t.Key("volume").MustInt(cfg.TTS.Volume)
	cfg.TTS.MaxQueue = t.Key("max_queue").MustInt(cfg.TTS.MaxQueue)
	cfg.TTS.MaxAgeSeconds = t.Key("max_age_seconds").MustInt(cfg.TTS.MaxAgeSeconds)
	cfg.TTS.TemplateText = t.Key("template_text").MustString(cfg.TTS.TemplateText)
	cfg.TTS.TemplateFreeGift = t.Key("template_free_gift").MustString(cfg.TTS.TemplateFreeGift)
	cfg.TTS.TemplatePaidGift = t.Key("template_paid_gift").MustString(cfg.TTS.TemplatePaidGift)
	cfg.TTS.TemplateMember = t.Key("template_member").MustString(cfg.TTS.TemplateMember)
	cfg.TTS.TemplateSuperChat = t.Key("template_super_chat").MustString(cfg.TTS.TemplateSuperChat)

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

	tt := f.Section("tts")
	tt.Key("enabled").SetValue(fmt.Sprintf("%t", normalized.TTS.Enabled))
	tt.Key("room_id").SetValue(fmt.Sprintf("%d", normalized.TTS.RoomID))
	tt.Key("voice_id").SetValue(normalized.TTS.VoiceID)
	tt.Key("rate").SetValue(fmt.Sprintf("%d", normalized.TTS.Rate))
	tt.Key("volume").SetValue(fmt.Sprintf("%d", normalized.TTS.Volume))
	tt.Key("max_queue").SetValue(fmt.Sprintf("%d", normalized.TTS.MaxQueue))
	tt.Key("max_age_seconds").SetValue(fmt.Sprintf("%d", normalized.TTS.MaxAgeSeconds))
	tt.Key("template_text").SetValue(normalized.TTS.TemplateText)
	tt.Key("template_free_gift").SetValue(normalized.TTS.TemplateFreeGift)
	tt.Key("template_paid_gift").SetValue(normalized.TTS.TemplatePaidGift)
	tt.Key("template_member").SetValue(normalized.TTS.TemplateMember)
	tt.Key("template_super_chat").SetValue(normalized.TTS.TemplateSuperChat)

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
	if normalized.TTS.Enabled {
		if normalized.TTS.RoomID <= 0 {
			return fmt.Errorf("tts.room_id must be a positive room ID when TTS is enabled")
		}
		if normalized.TTS.Rate < -10 || normalized.TTS.Rate > 10 {
			return fmt.Errorf("tts.rate must be between -10 and 10")
		}
		if normalized.TTS.Volume < 0 || normalized.TTS.Volume > 100 {
			return fmt.Errorf("tts.volume must be between 0 and 100")
		}
		if normalized.TTS.MaxQueue < 1 {
			return fmt.Errorf("tts.max_queue must be at least 1")
		}
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
	copy.TTS.VoiceID = strings.TrimSpace(copy.TTS.VoiceID)
	return copy
}
