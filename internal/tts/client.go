package tts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/iucario/bili-danmu-go/internal/chat"
)

// Client subscribes to the local SSE chat stream and dispatches TTS events.
type Client struct {
	baseURL string // e.g. "http://127.0.0.1:5090"
	roomID  atomic.Int64
	cfgPtr  atomic.Pointer[Config]
	queue   *TTSQueue
	// cancelConn holds the cancel func for the current SSE connection so that
	// UpdateRoomID can force an immediate reconnect with the new room ID.
	cancelConn atomic.Pointer[context.CancelFunc]
}

// NewClient creates a Client that will connect to the given server base URL.
func NewClient(baseURL string, roomID int64, cfg *Config, queue *TTSQueue) *Client {
	c := &Client{
		baseURL: baseURL,
		queue:   queue,
	}
	c.roomID.Store(roomID)
	c.cfgPtr.Store(cfg)
	return c
}

// UpdateConfig atomically replaces the running config (templates, etc.).
func (c *Client) UpdateConfig(cfg Config) {
	c.cfgPtr.Store(&cfg)
}

// UpdateRoomID changes the subscribed room and reconnects immediately.
func (c *Client) UpdateRoomID(id int64) {
	c.roomID.Store(id)
	if fn := c.cancelConn.Load(); fn != nil {
		(*fn)()
	}
}

// Run subscribes to the SSE stream and processes events, reconnecting on failure.
// It blocks until ctx is cancelled.
func (c *Client) Run(ctx context.Context) {
	slog.Info("tts/client: Run started")
	const (
		initialBackoff = time.Second
		maxBackoff     = 30 * time.Second
		disabledSleep  = 2 * time.Second
	)
	backoff := initialBackoff

	for {
		if !c.cfgPtr.Load().Enabled {
			slog.Info("tts/client: TTS disabled, waiting...")
			select {
			case <-ctx.Done():
				return
			case <-time.After(disabledSleep):
				continue
			}
		}

		connected, err := c.runOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			slog.Error("tts/client: connection error", "err", err)
		}
		if connected {
			backoff = initialBackoff
		} else {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
	}
}

// runOnce opens an SSE connection and reads until it closes or ctx is done.
// Returns (true, err) if the HTTP request succeeded, (false, err) on dial failure.
func (c *Client) runOnce(ctx context.Context) (connected bool, _ error) {
	connCtx, cancel := context.WithCancel(ctx)
	c.cancelConn.Store(&cancel)
	defer func() {
		cancel()
		c.cancelConn.Store(nil)
	}()

	roomID := c.roomID.Load()
	if roomID <= 0 {
		return false, fmt.Errorf("room ID not set, waiting for config")
	}
	url := fmt.Sprintf("%s/api/chat/stream?roomId=%d", c.baseURL, roomID)

	req, err := http.NewRequestWithContext(connCtx, http.MethodGet, url, nil)
	if err != nil {
		return false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("connect %s: %w", url, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return true, fmt.Errorf("unexpected status %d from %s", resp.StatusCode, url)
	}

	slog.Debug("tts/client: connecting to SSE stream", "roomId", roomID, "url", url)
	slog.Info("tts/client: connected to SSE stream", "roomId", roomID)
	return true, c.readLoop(connCtx, resp)
}

// readLoop parses the SSE text/event-stream from resp and dispatches events.
func (c *Client) readLoop(ctx context.Context, resp *http.Response) error {
	scanner := bufio.NewScanner(resp.Body)

	var eventName string
	var dataLines [][]byte

	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}
		line := scanner.Bytes()

		switch {
		case bytes.HasPrefix(line, []byte("event:")):
			eventName = string(bytes.TrimSpace(line[6:]))
		case bytes.HasPrefix(line, []byte("data:")):
			dataLines = append(dataLines, bytes.TrimSpace(line[5:]))
		case len(line) == 0:
			// Empty line = end of event block.
			if eventName != "" && len(dataLines) > 0 {
				data := bytes.Join(dataLines, nil)
				c.dispatch(eventName, data)
			}
			eventName = ""
			dataLines = dataLines[:0]
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("read SSE: %w", err)
	}
	return nil
}

func (c *Client) dispatch(event string, data []byte) {
	slog.Debug("tts/client: received event", "event", event, "data", truncate(string(data), 120))
	switch event {
	case "add_text":
		c.handleText(data)
	case "add_gift":
		c.handleGift(data)
	case "add_member":
		c.handleMember(data)
	case "add_super_chat":
		c.handleSuperChat(data)
	}
}

func (c *Client) handleText(data []byte) {
	var ev chat.AddTextEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		slog.Error("tts/client: unmarshal add_text", "err", err)
		return
	}
	if ev.IsGiftDanmaku || ev.ContentType != 0 {
		return
	}
	if ev.Content == "" {
		return
	}
	text := applyTemplate(c.cfgPtr.Load().TemplateText, map[string]string{
		"author_name": ev.AuthorName,
		"content":     ev.Content,
	})
	slog.Debug("tts/client: enqueue normal", "text", truncate(text, 80))
	c.queue.Enqueue(text, PriorityNormal, ev.Timestamp)
}

func (c *Client) handleGift(data []byte) {
	var ev chat.AddGiftEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		slog.Error("tts/client: unmarshal add_gift", "err", err)
		return
	}

	cfg := c.cfgPtr.Load()
	if ev.TotalCoin > 0 {
		price := float64(ev.TotalCoin) / 1000.0
		text := applyTemplate(cfg.TemplatePaidGift, map[string]string{
			"author_name": ev.AuthorName,
			"gift_name":   ev.GiftName,
			"num":         strconv.Itoa(ev.Num),
			"price":       fmt.Sprintf("%.1f", price),
		})
		c.queue.Enqueue(text, PriorityHigh, ev.Timestamp)
	} else {
		text := applyTemplate(cfg.TemplateFreeGift, map[string]string{
			"author_name": ev.AuthorName,
			"gift_name":   ev.GiftName,
			"num":         strconv.Itoa(ev.Num),
		})
		c.queue.Enqueue(text, PriorityNormal, ev.Timestamp)
	}
}

var guardNames = map[int]string{
	1: "总督",
	2: "提督",
	3: "舰长",
}

func (c *Client) handleMember(data []byte) {
	var ev chat.AddMemberEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		slog.Error("tts/client: unmarshal add_member", "err", err)
		return
	}

	guardName := guardNames[ev.PrivilegeType]
	if guardName == "" {
		guardName = "舰长"
	}

	text := applyTemplate(c.cfgPtr.Load().TemplateMember, map[string]string{
		"author_name": ev.AuthorName,
		"guard_name":  guardName,
	})
	c.queue.Enqueue(text, PriorityHigh, ev.Timestamp)
}

func (c *Client) handleSuperChat(data []byte) {
	var ev chat.AddSuperChatEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		slog.Error("tts/client: unmarshal add_super_chat", "err", err)
		return
	}

	text := applyTemplate(c.cfgPtr.Load().TemplateSuperChat, map[string]string{
		"author_name": ev.AuthorName,
		"price":       strconv.Itoa(ev.Price),
		"content":     ev.Content,
	})
	slog.Debug("tts/client: enqueue high(super_chat)", "text", truncate(text, 80))
	c.queue.Enqueue(text, PriorityHigh, ev.Timestamp)
}

// truncate shortens s to at most n runes for log display.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
