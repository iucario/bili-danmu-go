package bili

import (
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
)

const defaultEventBufSize = 64

// chanHandler routes every event to a buffered channel. It is the internal handler always used by BLiveClient.
type chanHandler struct {
	ch       chan Event
	fatalErr atomic.Pointer[error]
	once     sync.Once
}

func newChanHandler() *chanHandler {
	return &chanHandler{ch: make(chan Event, defaultEventBufSize)}
}

// send attempts a non-blocking write. If the buffer is full the event is
// dropped with a warning rather than blocking the WebSocket read loop.
func (h *chanHandler) send(ev Event) {
	select {
	case h.ch <- ev:
	default:
		slog.Warn("bili: event channel full, dropping event", "type", ev.Type)
	}
}

func (h *chanHandler) close() {
	h.once.Do(func() { close(h.ch) })
}

// OnFatalError satisfies the duck-typed interface checked by BLiveClient.
func (h *chanHandler) OnFatalError(err error) {
	h.fatalErr.Store(&err)
	h.close()
}

// dispatch parses a raw SEND_MSG_REPLY JSON body and sends the decoded event.
func (h *chanHandler) dispatch(raw []byte) {
	var msg RawMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		slog.Warn("bili: failed to unmarshal raw message", "err", err)
		return
	}

	// Strip `:` suffix (e.g. "DANMU_MSG:4:0:2:2:2:0" → "DANMU_MSG")
	cmd := msg.Cmd
	if idx := strings.Index(cmd, ":"); idx != -1 {
		cmd = cmd[:idx]
	}

	switch cmd {
	case "DANMU_MSG":
		info, err := ParseDanmakuInfo(msg.Info)
		if err != nil {
			slog.Warn("bili: parse danmaku info", "err", err)
			return
		}
		slog.Debug("bili: danmaku", "user", info.Uname, "msg", info.Msg, "lottery", info.NotShow)
		h.send(Event{Type: EventTypeDanmaku, Danmaku: info})

	case "DANMU_MSG_MIRROR":
		info, err := ParseDanmakuInfo(msg.Info)
		if err != nil {
			slog.Warn("bili: parse danmaku mirror info", "err", err)
			return
		}
		info.IsMirror = true
		slog.Debug("bili: danmaku_mirror", "user", info.Uname, "msg", info.Msg)
		h.send(Event{Type: EventTypeDanmaku, Danmaku: info})

	case "SEND_GIFT":
		var data GiftData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			slog.Warn("bili: parse gift data", "err", err)
			return
		}
		slog.Debug("bili: gift", "user", data.Uname, "gift", data.GiftName, "num", data.Num)
		h.send(Event{Type: EventTypeGift, Gift: &data})

	case "USER_TOAST_MSG_V2":
		var data UserToastV2Data
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			slog.Warn("bili: parse user_toast_v2 data", "err", err)
			return
		}
		slog.Debug("bili: guard", "user", data.SenderUinfo.Base.Name, "level", data.GuardInfo.GuardLevel)
		h.send(Event{Type: EventTypeUserToastV2, UserToastV2: &data})

	case "SUPER_CHAT_MESSAGE":
		var data SuperChatData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			slog.Warn("bili: parse super_chat data", "err", err)
			return
		}
		slog.Debug("bili: superchat", "user", data.UserInfo.Uname, "price", data.Price, "msg", data.Message)
		h.send(Event{Type: EventTypeSuperChat, SuperChat: &data})

	case "SUPER_CHAT_MESSAGE_DELETE":
		var data SuperChatDeleteData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			slog.Warn("bili: parse super_chat_delete data", "err", err)
			return
		}
		slog.Debug("bili: superchat_delete", "ids", data.IDs)
		h.send(Event{Type: EventTypeSuperChatDelete, SuperChatDelete: &data})

	default:
		// Silently ignore unhandled commands (INTERACT_WORD_V2, WATCHED_CHANGE, etc.)
	}
}
