package chat

import (
	"crypto/rand"
	"fmt"
)

func newID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return fmt.Sprintf("%x", b)
}

// AddTextEvent is the SSE add_text payload.
type AddTextEvent struct {
	ID                string            `json:"id"`
	Timestamp         int64             `json:"timestamp"`
	AuthorName        string            `json:"authorName"`
	AuthorType        int               `json:"authorType"` // 0=normal 1=guard 2=admin 3=owner
	Content           string            `json:"content"`
	PrivilegeType     int               `json:"privilegeType"` // 0=none 1=总督 2=提督 3=舰长
	IsGiftDanmaku     bool              `json:"isGiftDanmaku"`
	IsLottery         bool              `json:"isLottery"`
	AuthorLevel       int               `json:"authorLevel"`
	IsNewbie          bool              `json:"isNewbie"`
	IsMobileVerified  bool              `json:"isMobileVerified"`
	MedalLevel        int               `json:"medalLevel"`
	MedalName         string            `json:"medalName"`
	AvatarURL         string            `json:"avatarUrl"`
	UID               string            `json:"uid"`
	ContentType       int               `json:"contentType"` // 0=text 1=emoticon
	ContentTypeParams map[string]string `json:"contentTypeParams"`
	IsMirror          bool              `json:"isMirror"`
	Translation       string            `json:"translation"`
}

// AddGiftEvent is the SSE add_gift payload.
type AddGiftEvent struct {
	ID            string `json:"id"`
	Timestamp     int64  `json:"timestamp"`
	AuthorName    string `json:"authorName"`
	AvatarURL     string `json:"avatarUrl"`
	UID           string `json:"uid"`
	GiftName      string `json:"giftName"`
	Num           int    `json:"num"`
	TotalCoin     int    `json:"totalCoin"`
	TotalFreeCoin int    `json:"totalFreeCoin"`
	GiftID        int    `json:"giftId"`
	GiftIconURL   string `json:"giftIconUrl"`
	PrivilegeType int    `json:"privilegeType"`
	MedalLevel    int    `json:"medalLevel"`
	MedalName     string `json:"medalName"`
}

// AddMemberEvent is the SSE add_member payload (guard purchase).
type AddMemberEvent struct {
	ID            string `json:"id"`
	Timestamp     int64  `json:"timestamp"`
	AuthorName    string `json:"authorName"`
	AvatarURL     string `json:"avatarUrl"`
	UID           string `json:"uid"`
	PrivilegeType int    `json:"privilegeType"`
	Num           int    `json:"num"`
	Unit          string `json:"unit"`
	TotalCoin     int    `json:"totalCoin"`
	MedalLevel    int    `json:"medalLevel"`
	MedalName     string `json:"medalName"`
}

// AddSuperChatEvent is the SSE add_super_chat payload.
type AddSuperChatEvent struct {
	ID            string `json:"id"`
	Timestamp     int64  `json:"timestamp"`
	AuthorName    string `json:"authorName"`
	AvatarURL     string `json:"avatarUrl"`
	UID           string `json:"uid"`
	Price         int    `json:"price"`
	Content       string `json:"content"`
	Translation   string `json:"translation"`
	PrivilegeType int    `json:"privilegeType"`
	MedalLevel    int    `json:"medalLevel"`
	MedalName     string `json:"medalName"`
}

// DelSuperChatEvent is the SSE del_super_chat payload.
type DelSuperChatEvent struct {
	IDs []string `json:"ids"`
}

// ChatEvent is a typed in-process event emitted by a room.
// Exactly one pointer field is non-nil, identified by Type.
type ChatEvent struct {
	Type      string             // "add_text" | "add_gift" | "add_member" | "add_super_chat"
	Text      *AddTextEvent
	Gift      *AddGiftEvent
	Member    *AddMemberEvent
	SuperChat *AddSuperChatEvent
}

// payload returns the non-nil event struct for JSON marshaling.
func (e ChatEvent) payload() any {
	switch e.Type {
	case "add_text":
		return e.Text
	case "add_gift":
		return e.Gift
	case "add_member":
		return e.Member
	case "add_super_chat":
		return e.SuperChat
	}
	return nil
}

// FatalErrorEvent is sent when the Bilibili connection is unrecoverable.
type FatalErrorEvent struct {
	Type string `json:"type"`
	Msg  string `json:"msg"`
}
