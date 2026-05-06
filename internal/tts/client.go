package tts

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/iucario/bili-danmu-go/internal/chat"
)

// Client subscribes to the in-process chat event stream and dispatches TTS events.
// No HTTP or SSE parsing — events are delivered via a Go channel directly from the broadcaster.
type Client struct {
	rm     *chat.RoomManager
	roomID atomic.Int64
	cfgPtr atomic.Pointer[Config]
	queue  *TTSQueue
	// cancelConn cancels the current room subscription so UpdateRoomID can
	// force an immediate resubscription with the new room ID.
	cancelConn atomic.Pointer[context.CancelFunc]
}

// NewClient creates a Client backed by the given RoomManager.
func NewClient(rm *chat.RoomManager, roomID int64, cfg *Config, queue *TTSQueue) *Client {
	c := &Client{rm: rm, queue: queue}
	c.roomID.Store(roomID)
	c.cfgPtr.Store(cfg)
	return c
}

// UpdateConfig atomically replaces the running config.
// If TTS is being disabled, the current subscription is cancelled immediately.
func (c *Client) UpdateConfig(cfg Config) {
	prev := c.cfgPtr.Swap(&cfg)
	if prev != nil && prev.Enabled && !cfg.Enabled {
		if fn := c.cancelConn.Load(); fn != nil {
			(*fn)()
		}
	}
}

// UpdateRoomID changes the subscribed room and resubscribes immediately.
func (c *Client) UpdateRoomID(id int64) {
	c.roomID.Store(id)
	if fn := c.cancelConn.Load(); fn != nil {
		(*fn)()
	}
}

// Run subscribes to room events and dispatches them to the TTS queue.
// It blocks until ctx is cancelled.
func (c *Client) Run(ctx context.Context) {
	slog.Info("tts/client: Run started")
	const disabledSleep = 2 * time.Second

	for {
		if !c.cfgPtr.Load().Enabled {
			select {
			case <-ctx.Done():
				return
			case <-time.After(disabledSleep):
				continue
			}
		}

		roomID := c.roomID.Load()
		if roomID <= 0 {
			slog.Warn("tts/client: room ID not set, waiting for config")
			select {
			case <-ctx.Done():
				return
			case <-time.After(disabledSleep):
				continue
			}
		}

		subCtx, cancel := context.WithCancel(ctx)
		c.cancelConn.Store(&cancel)

		events, unsub := c.rm.SubscribeEvents(subCtx, roomID)
		slog.Info("tts/client: subscribed to room events", "roomId", roomID)

		c.processEvents(subCtx, events)

		unsub()
		cancel()
		c.cancelConn.Store(nil)

		if ctx.Err() != nil {
			return
		}
	}
}

// processEvents reads from the event channel until ctx is cancelled or TTS is disabled.
func (c *Client) processEvents(ctx context.Context, events <-chan chat.ChatEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-events:
			if !ok {
				return
			}
			if !c.cfgPtr.Load().Enabled {
				return
			}
			c.dispatch(ev)
		}
	}
}

func (c *Client) dispatch(ev chat.ChatEvent) {
	slog.Debug("tts/client: received event", "type", ev.Type)
	switch ev.Type {
	case "add_text":
		c.handleText(ev.Text)
	case "add_gift":
		c.handleGift(ev.Gift)
	case "add_member":
		c.handleMember(ev.Member)
	case "add_super_chat":
		c.handleSuperChat(ev.SuperChat)
	}
}

func (c *Client) handleText(ev *chat.AddTextEvent) {
	if ev.IsGiftDanmaku || ev.ContentType != 0 || ev.Content == "" {
		return
	}
	text := applyTemplate(c.cfgPtr.Load().TemplateText, map[string]string{
		"author_name": ev.AuthorName,
		"content":     ev.Content,
	})
	slog.Debug("tts/client: enqueue normal", "text", truncate(text, 80))
	c.queue.Enqueue(text, PriorityNormal, ev.Timestamp)
}

func (c *Client) handleGift(ev *chat.AddGiftEvent) {
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

func (c *Client) handleMember(ev *chat.AddMemberEvent) {
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

func (c *Client) handleSuperChat(ev *chat.AddSuperChatEvent) {
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
