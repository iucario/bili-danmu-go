package bili

import (
	"encoding/json"
	"log/slog"
	"strings"
)

// HandlerInterface is implemented by types that want to receive
// decoded Bilibili live messages.
type HandlerInterface interface {
	OnDanmaku(info *DanmakuInfo)
	OnGift(data *GiftData)
	OnUserToastV2(data *UserToastV2Data)
	OnSuperChat(data *SuperChatData)
	OnSuperChatDelete(data *SuperChatDeleteData)
}

// BaseHandler dispatches raw SEND_MSG_REPLY business messages to a HandlerInterface.
// Embed or wrap it to receive decoded events.
type BaseHandler struct {
	Handler HandlerInterface
}

// Dispatch parses a raw JSON business message and routes it to the handler.
func (b *BaseHandler) Dispatch(raw []byte) {
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
		b.Handler.OnDanmaku(info)

	case "DANMU_MSG_MIRROR":
		info, err := ParseDanmakuInfo(msg.Info)
		if err != nil {
			slog.Warn("bili: parse danmaku mirror info", "err", err)
			return
		}
		info.IsMirror = true
		b.Handler.OnDanmaku(info)

	case "SEND_GIFT":
		var data GiftData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			slog.Warn("bili: parse gift data", "err", err)
			return
		}
		b.Handler.OnGift(&data)

	case "USER_TOAST_MSG_V2":
		var data UserToastV2Data
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			slog.Warn("bili: parse user_toast_v2 data", "err", err)
			return
		}
		b.Handler.OnUserToastV2(&data)

	case "SUPER_CHAT_MESSAGE":
		var data SuperChatData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			slog.Warn("bili: parse super_chat data", "err", err)
			return
		}
		b.Handler.OnSuperChat(&data)

	case "SUPER_CHAT_MESSAGE_DELETE":
		var data SuperChatDeleteData
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			slog.Warn("bili: parse super_chat_delete data", "err", err)
			return
		}
		b.Handler.OnSuperChatDelete(&data)

	default:
		// Silently ignore unhandled commands (INTERACT_WORD_V2, etc.)
	}
}
