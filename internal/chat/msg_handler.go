package chat

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"sync/atomic"

	"github.com/iucario/danmu-go/internal/bili"
)

// LiveMsgHandler translates raw Bilibili messages into SSE events and broadcasts
// them to the associated ClientRoom. It implements bili.HandlerInterface.
type LiveMsgHandler struct {
	room   *ClientRoom
	roomID int64 // original (short) room ID supplied by the user

	realRoomID atomic.Int64
	ownerUID   atomic.Int64

	// onFatal is called when the connection is unrecoverable (too many retries).
	onFatal func()
}

func NewLiveMsgHandler(roomID int64, room *ClientRoom, onFatal func()) *LiveMsgHandler {
	return &LiveMsgHandler{
		room:    room,
		roomID:  roomID,
		onFatal: onFatal,
	}
}

// OnConnect is called by BLiveClient after a successful room info fetch.
func (h *LiveMsgHandler) OnConnect(realRoomID, ownerUID int64) {
	h.realRoomID.Store(realRoomID)
	h.ownerUID.Store(ownerUID)
}

// OnFatalError satisfies the duck-typed interface checked by BLiveClient.
func (h *LiveMsgHandler) OnFatalError(_ error) {
	h.broadcast("fatal_error", FatalErrorEvent{
		Type: "too_many_retries",
		Msg:  "The connection has been lost too many times",
	})
	if h.onFatal != nil {
		h.onFatal()
	}
}

// ---- bili.HandlerInterface ----

func (h *LiveMsgHandler) OnDanmaku(info *bili.DanmakuInfo) {
	content := info.Msg

	// Prepend reply username if present in mode_info.extra.
	if len(info.ModeInfo) > 0 {
		var modeInfo struct {
			Extra string `json:"extra"`
			User  struct {
				Base struct {
					Face string `json:"face"`
				} `json:"base"`
			} `json:"user"`
		}
		if err := json.Unmarshal(info.ModeInfo, &modeInfo); err == nil {
			if modeInfo.Extra != "" {
				var extra struct {
					ReplyUname string `json:"reply_uname"`
				}
				if err := json.Unmarshal([]byte(modeInfo.Extra), &extra); err == nil && extra.ReplyUname != "" {
					content = fmt.Sprintf("回复 @%s: %s", extra.ReplyUname, content)
				}
			}
		}
	}

	// Avatar URL from mode_info.user.base.face
	avatarURL := h.avatarFromModeInfo(info.ModeInfo)

	// Medal: only show if it belongs to this room.
	medalLevel, medalName := 0, ""
	if info.MedalRoomID == h.realRoomID.Load() {
		medalLevel = info.MedalLevel
		medalName = info.MedalName
	}

	// ContentType and emoticon URL.
	contentType := 0
	contentTypeParams := map[string]string{}
	if info.MsgType == 1 {
		contentType = 1
		if len(info.EmoticonOptions) > 0 {
			var emo struct {
				URL string `json:"url"`
			}
			if err := json.Unmarshal(info.EmoticonOptions, &emo); err == nil && emo.URL != "" {
				contentTypeParams["url"] = emo.URL
			}
		}
	}

	ev := AddTextEvent{
		ID:                newID(),
		Timestamp:         info.Timestamp,
		AuthorName:        info.Uname,
		AuthorType:        h.authorType(info),
		Content:           content,
		PrivilegeType:     info.PrivilegeType,
		IsGiftDanmaku:     info.DmType == 1,
		IsLottery:         info.NotShow,
		AuthorLevel:       info.UserLevel,
		IsNewbie:          info.URank < 10000,
		IsMobileVerified:  info.MobileVerify == 1,
		MedalLevel:        medalLevel,
		MedalName:         medalName,
		AvatarURL:         avatarURL,
		UID:               strconv.FormatInt(info.UID, 10),
		ContentType:       contentType,
		ContentTypeParams: contentTypeParams,
		IsMirror:          info.IsMirror,
	}
	h.broadcast("add_text", ev)
}

func (h *LiveMsgHandler) OnGift(data *bili.GiftData) {
	totalCoin, totalFreeCoin := 0, 0
	if data.CoinType == "gold" {
		totalCoin = data.TotalCoin
	} else {
		totalFreeCoin = data.TotalCoin
	}

	medalLevel, medalName := 0, ""
	if data.MedalInfo.AnchorRoomID == h.realRoomID.Load() {
		medalLevel = data.MedalInfo.MedalLevel
		medalName = data.MedalInfo.MedalName
	}

	ev := AddGiftEvent{
		ID:            newID(),
		Timestamp:     data.Timestamp,
		AuthorName:    data.Uname,
		AvatarURL:     data.Face,
		UID:           strconv.FormatInt(data.UID, 10),
		GiftName:      data.GiftName,
		Num:           data.Num,
		TotalCoin:     totalCoin,
		TotalFreeCoin: totalFreeCoin,
		GiftID:        data.GiftID,
		GiftIconURL:   data.GiftInfo.ImgBasic,
		PrivilegeType: data.GuardLevel,
		MedalLevel:    medalLevel,
		MedalName:     medalName,
	}
	h.broadcast("add_gift", ev)
}

func (h *LiveMsgHandler) OnUserToastV2(data *bili.UserToastV2Data) {
	// source==2 means duplicate/gift message — skip.
	if data.Option.Source == 2 {
		return
	}

	medalLevel, medalName := 0, ""
	// UserToastV2 doesn't carry medal_info directly; leave zeroed.

	ev := AddMemberEvent{
		ID:            newID(),
		Timestamp:     data.GuardInfo.StartTime,
		AuthorName:    data.SenderUinfo.Base.Name,
		AvatarURL:     data.SenderUinfo.Base.Face,
		UID:           strconv.FormatInt(data.SenderUinfo.UID, 10),
		PrivilegeType: data.GuardInfo.GuardLevel,
		Num:           data.PayInfo.Num,
		Unit:          data.PayInfo.Unit,
		TotalCoin:     data.PayInfo.Price * data.PayInfo.Num,
		MedalLevel:    medalLevel,
		MedalName:     medalName,
	}
	h.broadcast("add_member", ev)
}

func (h *LiveMsgHandler) OnSuperChat(data *bili.SuperChatData) {
	medalLevel, medalName := 0, ""
	if data.MedalInfo.AnchorRoomID == h.realRoomID.Load() {
		medalLevel = data.MedalInfo.MedalLevel
		medalName = data.MedalInfo.MedalName
	}

	ev := AddSuperChatEvent{
		ID:            newID(),
		Timestamp:     data.StartTime,
		AuthorName:    data.UserInfo.Uname,
		AvatarURL:     data.UserInfo.Face,
		UID:           data.UID.String(),
		Price:         data.Price,
		Content:       data.Message,
		Translation:   data.MessageTrans,
		PrivilegeType: data.UserInfo.GuardLevel,
		MedalLevel:    medalLevel,
		MedalName:     medalName,
	}
	h.broadcast("add_super_chat", ev)
}

func (h *LiveMsgHandler) OnSuperChatDelete(data *bili.SuperChatDeleteData) {
	ids := make([]string, len(data.IDs))
	for i, id := range data.IDs {
		ids[i] = strconv.FormatInt(id, 10)
	}
	h.broadcast("del_super_chat", DelSuperChatEvent{IDs: ids})
}

// ---- helpers ----

func (h *LiveMsgHandler) authorType(info *bili.DanmakuInfo) int {
	if info.UID == h.ownerUID.Load() && h.ownerUID.Load() != 0 {
		return 3
	}
	if info.Admin == 1 {
		return 2
	}
	if info.PrivilegeType > 0 {
		return 1
	}
	return 0
}

func (h *LiveMsgHandler) avatarFromModeInfo(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var mi struct {
		User struct {
			Base struct {
				Face string `json:"face"`
			} `json:"base"`
		} `json:"user"`
	}
	if err := json.Unmarshal(raw, &mi); err != nil {
		return ""
	}
	return mi.User.Base.Face
}

func (h *LiveMsgHandler) broadcast(eventName string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		slog.Warn("chat: marshal event", "event", eventName, "err", err)
		return
	}
	frame := fmt.Appendf(nil, "event: %s\ndata: %s\n\n", eventName, b)
	h.room.Broadcast(frame)
}
