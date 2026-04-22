package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/iucario/danmu-go/internal/chat"
)

// NewChatHandler returns an http.Handler for GET /api/chat/stream.
func NewChatHandler(rm *chat.RoomManager) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roomIDStr := r.URL.Query().Get("roomId")
		roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
		if err != nil || roomID <= 0 {
			http.Error(w, "invalid roomId", http.StatusBadRequest)
			return
		}

		// Disable write deadline for long-lived SSE connection.
		rc := http.NewResponseController(w)
		if err := rc.SetWriteDeadline(time.Time{}); err != nil {
			slog.WarnContext(r.Context(), "failed to disable write deadline", "err", err)
		}

		conn, err := newSSEConn(w)
		if err != nil {
			slog.ErrorContext(r.Context(), "SSE not supported", "err", err)
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		ch, unsub := rm.Subscribe(r.Context(), roomID)
		defer unsub()

		slog.InfoContext(r.Context(), "SSE subscriber connected", "roomID", roomID)
		conn.start(r.Context(), ch)
		slog.InfoContext(r.Context(), "SSE subscriber disconnected", "roomID", roomID)
	})
}

// sseConn wraps an http.ResponseWriter for SSE writes.
type sseConn struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func newSSEConn(w http.ResponseWriter) (*sseConn, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("ResponseWriter does not implement http.Flusher")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering
	return &sseConn{w: w, flusher: flusher}, nil
}

// start blocks until the context is cancelled or a write error occurs.
// It drains ch and sends keepalive comments every 20 seconds.
func (c *sseConn) start(ctx context.Context, ch <-chan []byte) {
	keepalive := time.NewTicker(20 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case frame, ok := <-ch:
			if !ok {
				return
			}
			if err := c.write(frame); err != nil {
				return
			}
		case <-keepalive.C:
			if err := c.comment("keepalive"); err != nil {
				return
			}
		}
	}
}

func (c *sseConn) write(frame []byte) error {
	if _, err := c.w.Write(frame); err != nil {
		return fmt.Errorf("write SSE frame: %w", err)
	}
	c.flusher.Flush()
	return nil
}

func (c *sseConn) comment(s string) error {
	if _, err := fmt.Fprintf(c.w, ": %s\n\n", s); err != nil {
		return fmt.Errorf("write SSE comment: %w", err)
	}
	c.flusher.Flush()
	return nil
}
