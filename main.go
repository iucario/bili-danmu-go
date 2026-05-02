package main

import (
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/iucario/bili-danmu-go/internal/api"
	"github.com/iucario/bili-danmu-go/internal/chat"
	"github.com/iucario/bili-danmu-go/internal/config"
	"github.com/iucario/bili-danmu-go/internal/version"
	"github.com/iucario/bili-danmu-go/pkg/bili"
	"github.com/iucario/bili-danmu-go/server"
)

//go:embed internal/api/index.html
var indexFS embed.FS

func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to relative path if the OS doesn't provide a config dir.
		return filepath.Join("data", "config.ini")
	}
	return filepath.Join(dir, "bili-danmu-go", "config.ini")
}

func main() {
	index, err := fs.Sub(indexFS, "internal/api")
	if err != nil {
		slog.Error("failed to read embedded index.html", "err", err)
		os.Exit(1)
	}
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
	cs := config.New(*configPath, cfg, func(prev, next *config.Config) error {
		if prev.SESSDATA != next.SESSDATA {
			bili.SetSESSDATA(next.SESSDATA)
		}
		return nil
	})

	mux := http.NewServeMux()
	mux.Handle("GET /api/chat/stream", api.NewChatHandler(rm))
	mux.Handle("GET /api/avatar", api.NewAvatarProxyHandler())
	mux.Handle("/api/config", api.NewConfigHandler(cs))
	mux.Handle("GET /obs/", obsHandler())
	mux.Handle("GET /admin/", adminHandler())

	mux.Handle("GET /{$}", http.FileServer(http.FS(index)))

	if err := server.Run(cfg, mux, rm.StopAll); err != nil {
		slog.Error("server stopped with error", "err", err)
		fmt.Fprintf(os.Stderr, "\nERROR: %v\n", err)
		os.Exit(1)
	}
	slog.Info("goodbye")
}

// bestEffortWriter silently discards write errors. Used to wrap os.Stderr so that a failed stderr write (e.g. on Windows GUI builds with no console) does not prevent writes to other io.MultiWriter targets.
type bestEffortWriter struct{ w io.Writer }

func (b *bestEffortWriter) Write(p []byte) (int, error) {
	_, _ = b.w.Write(p)
	return len(p), nil
}

// openLogFile creates (or appends to) bili-danmu-go.log in the OS temp directory and configures slog to write to both stderr and the file. Returns the file so the caller can defer-close it; returns nil on failure.
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

	// Wrap stderr in a best-effort writer: on Windows GUI builds (-H windowsgui) os.Stderr is an invalid handle, and io.MultiWriter stops on the first error — meaning the file would never be written either. Silently dropping stderr errors lets the file writer always succeed.
	w := io.MultiWriter(&bestEffortWriter{os.Stderr}, f)
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
