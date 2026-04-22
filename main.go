package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/iucario/danmu-go/api"
	"github.com/iucario/danmu-go/config"
	"github.com/iucario/danmu-go/internal/chat"
	"github.com/iucario/danmu-go/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg := config.Default()
	rm := chat.NewRoomManager()

	mux := http.NewServeMux()
	mux.Handle("GET /api/chat/stream", api.NewChatHandler(rm))

	if err := server.Run(cfg, mux, rm.StopAll); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
