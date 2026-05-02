package bili

import (
	"encoding/json"
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
	return newChanHandlerSized(defaultEventBufSize)
}

func newChanHandlerSized(size int) *chanHandler {
	return &chanHandler{ch: make(chan Event, size)}
}

// send attempts a non-blocking write; drops the event if the buffer is full.
func (h *chanHandler) send(ev Event) {
	select {
	case h.ch <- ev:
	default:
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
			return
		}
		h.send(Event{Type: EventTypeDanmaku, Danmaku: info})

	case "DANMU_MSG_MIRROR":
		info, err := ParseDanmakuInfo(msg.Info)
		if err != nil {
			return
		}
		info.IsMirror = true
		h.send(Event{Type: EventTypeDanmaku, Danmaku: info})

	case "SEND_GIFT":
		var data GiftData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		h.send(Event{Type: EventTypeGift, Gift: &data})

	case "USER_TOAST_MSG_V2":
		var data UserToastV2Data
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		h.send(Event{Type: EventTypeUserToastV2, UserToastV2: &data})

	case "SUPER_CHAT_MESSAGE":
		var data SuperChatData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		h.send(Event{Type: EventTypeSuperChat, SuperChat: &data})

	case "SUPER_CHAT_MESSAGE_DELETE":
		var data SuperChatDeleteData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}
		h.send(Event{Type: EventTypeSuperChatDelete, SuperChatDelete: &data})

	default:
		// Silently ignore unhandled commands (INTERACT_WORD_V2, WATCHED_CHANGE, etc.)
	}
}
