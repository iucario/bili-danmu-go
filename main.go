package main

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/iucario/bili-danmu-go/api"
	"github.com/iucario/bili-danmu-go/config"
	"github.com/iucario/bili-danmu-go/internal/appconfig"
	"github.com/iucario/bili-danmu-go/internal/bili"
	"github.com/iucario/bili-danmu-go/internal/chat"
	"github.com/iucario/bili-danmu-go/internal/version"
	"github.com/iucario/bili-danmu-go/server"
)

func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to relative path if the OS doesn't provide a config dir.
		return filepath.Join("data", "config.ini")
	}
	return filepath.Join(dir, "bili-danmu-go", "config.ini")
}

func main() {
	configPath := flag.String("config", defaultConfigPath(),
		"path to config file (created with defaults if missing)")
	flag.Parse()

	// Load (or create) config first — we need the log level before setting up logging.
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "err", err, "path", *configPath)
		os.Exit(1)
	}

	// Set up logging to stderr + a log file.
	logFile := openLogFile(cfg.LogLevel)
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
	}
	slog.Info("config loaded", "path", *configPath, "log_level", cfg.LogLevel, "version", version.Version)

	bili.SetSESSDATA(cfg.SESSDATA)

	rm := chat.NewRoomManager()
	cs := appconfig.New(*configPath, cfg, func(prev, next *config.Config) error {
		if prev.SESSDATA != next.SESSDATA {
			bili.SetSESSDATA(next.SESSDATA)
		}
		return nil
	})

	mux := http.NewServeMux()
	mux.Handle("GET /api/chat/stream", api.NewChatHandler(rm))
	mux.Handle("GET /api/avatar", api.NewAvatarProxyHandler())
	mux.Handle("/api/config", api.NewConfigHandler(cs))
	mux.Handle("/obs/", obsHandler())
	mux.Handle("/admin/", adminHandler())

	if err := server.Run(cfg, mux, rm.StopAll); err != nil {
		slog.Error("server stopped with error", "err", err)
		fmt.Fprintf(os.Stderr, "\nERROR: %v\n", err)
		os.Exit(1)
	}
	slog.Info("goodbye")
}

// openLogFile creates (or appends to) bili-danmu-go.log in the OS temp directory
// and configures slog to write to both stderr and the file.
// Returns the file so the caller can defer-close it; returns nil on failure.
//
// Temp directory locations:
//   - Linux/macOS: /tmp
//   - Windows:     %TEMP% (e.g. C:\Users\<user>\AppData\Local\Temp)
func openLogFile(levelStr string) *os.File {
	var level slog.Level
	switch levelStr {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	logPath := filepath.Join(os.TempDir(), "bili-danmu-go.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Warn("could not open log file", "path", logPath, "err", err)
		return nil
	}

	w := io.MultiWriter(os.Stderr, f)
	slog.SetDefault(slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			// Format timestamps as human-readable local time.
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(time.Now().Format("2006-01-02 15:04:05"))
			}
			return a
		},
	})))

	slog.Info("logging to file", "path", logPath)
	return f
}
