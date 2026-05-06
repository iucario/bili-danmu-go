package chat

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/iucario/bili-danmu-go/pkg/bili"
)

// startEventLoop wires up OnConnect on client, then starts a goroutine that reads events from client.Events(), converts them to SSE frames and broadcasts to room.
// When the event channel closes the goroutine checks client.Err(); if it is non-nil it broadcasts a fatal_error SSE frame and calls onFatal.
func startEventLoop(client bili.Client, room *ClientRoom, onFatal func()) {
	go func() {
		for ev := range client.Events() {
			switch ev.Type {
			case bili.EventTypeDanmaku:
				handleDanmaku(ev.Danmaku, room, client.RealRoomID(), client.OwnerUID())
			case bili.EventTypeGift:
				handleGift(ev.Gift, room, client.RealRoomID())
			case bili.EventTypeUserToastV2:
				handleUserToastV2(ev.UserToastV2, room)
			case bili.EventTypeSuperChat:
				handleSuperChat(ev.SuperChat, room, client.RealRoomID())
			case bili.EventTypeSuperChatDelete:
				handleSuperChatDelete(ev.SuperChatDelete, room)
			}
		}
		if err := client.Err(); err != nil {
			broadcastSSE(room, "fatal_error", FatalErrorEvent{
				Type: "too_many_retries",
				Msg:  "The connection has been lost too many times",
			})
			if onFatal != nil {
				onFatal()
			}
		}
	}()
}

func handleDanmaku(info *bili.DanmakuInfo, room *ClientRoom, realRoomID, ownerUID int64) {
	content := info.Msg

	if len(info.ModeInfo) > 0 {
		var modeInfo struct {
			Extra string `json:"extra"`
			User  struct {
				Base struct {
					Face string `json:"face"`
				} `json:"base"`
			} `json:"user"`
		}
		if err := json.Unmarshal(info.ModeInfo, &modeInfo); err == nil && modeInfo.Extra != "" {
			var extra struct {
				ReplyUname string `json:"reply_uname"`
			}
			if err := json.Unmarshal([]byte(modeInfo.Extra), &extra); err == nil && extra.ReplyUname != "" {
				content = fmt.Sprintf("回复 @%s: %s", extra.ReplyUname, content)
			}
		}
	}

	avatarURL := avatarFromModeInfo(info.ModeInfo)

	medalLevel, medalName := 0, ""
	if info.MedalRoomID == realRoomID {
		medalLevel = info.MedalLevel
		medalName = info.MedalName
	}

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
		AuthorType:        authorType(info, ownerUID),
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
	broadcastEvent(room, ChatEvent{Type: "add_text", Text: &ev})
}

func handleGift(data *bili.GiftData, room *ClientRoom, realRoomID int64) {
	totalCoin, totalFreeCoin := 0, 0
	if data.CoinType == "gold" {
		totalCoin = data.TotalCoin
	} else {
		totalFreeCoin = data.TotalCoin
	}

	medalLevel, medalName := 0, ""
	if data.MedalInfo.AnchorRoomID == realRoomID {
		medalLevel = data.MedalInfo.MedalLevel
		medalName = data.MedalInfo.MedalName
	}

	giftEv := AddGiftEvent{
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
	broadcastEvent(room, ChatEvent{Type: "add_gift", Gift: &giftEv})
}

func handleUserToastV2(data *bili.UserToastV2Data, room *ClientRoom) {
	// source==2 means duplicate/gift message — skip.
	if data.Option.Source == 2 {
		return
	}
	memberEv := AddMemberEvent{
		ID:            newID(),
		Timestamp:     data.GuardInfo.StartTime,
		AuthorName:    data.SenderUinfo.Base.Name,
		AvatarURL:     data.SenderUinfo.Base.Face,
		UID:           strconv.FormatInt(data.SenderUinfo.UID, 10),
		PrivilegeType: data.GuardInfo.GuardLevel,
		Num:           data.PayInfo.Num,
		Unit:          data.PayInfo.Unit,
		TotalCoin:     data.PayInfo.Price * data.PayInfo.Num,
	}
	broadcastEvent(room, ChatEvent{Type: "add_member", Member: &memberEv})
}

func handleSuperChat(data *bili.SuperChatData, room *ClientRoom, realRoomID int64) {
	medalLevel, medalName := 0, ""
	if data.MedalInfo.AnchorRoomID == realRoomID {
		medalLevel = data.MedalInfo.MedalLevel
		medalName = data.MedalInfo.MedalName
	}
	scEv := AddSuperChatEvent{
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
	broadcastEvent(room, ChatEvent{Type: "add_super_chat", SuperChat: &scEv})
}

func handleSuperChatDelete(data *bili.SuperChatDeleteData, room *ClientRoom) {
	ids := make([]string, len(data.IDs))
	for i, id := range data.IDs {
		ids[i] = strconv.FormatInt(id, 10)
	}
	broadcastSSE(room, "del_super_chat", DelSuperChatEvent{IDs: ids})
}

func authorType(info *bili.DanmakuInfo, ownerUID int64) int {
	if ownerUID != 0 && info.UID == ownerUID {
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

func avatarFromModeInfo(raw []byte) string {
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

func broadcastEvent(room *ClientRoom, ev ChatEvent) {
	room.BroadcastEvent(ev)
	b, err := json.Marshal(ev.payload())
	if err != nil {
		slog.Warn("chat: marshal event", "event", ev.Type, "err", err)
		return
	}
	room.Broadcast(fmt.Appendf(nil, "event: %s\ndata: %s\n\n", ev.Type, b))
}

// broadcastSSE sends a raw SSE-only event (no in-process typed delivery).
func broadcastSSE(room *ClientRoom, eventName string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		slog.Warn("chat: marshal event", "event", eventName, "err", err)
		return
	}
	room.Broadcast(fmt.Appendf(nil, "event: %s\ndata: %s\n\n", eventName, b))
}
