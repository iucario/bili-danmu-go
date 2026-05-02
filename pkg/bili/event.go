package bili

// Event type constants.
const (
	EventTypeDanmaku         = "danmaku"
	EventTypeGift            = "gift"
	EventTypeUserToastV2     = "user_toast_v2"
	EventTypeSuperChat       = "super_chat"
	EventTypeSuperChatDelete = "super_chat_delete"
)

// Event is a discriminated union delivered on the channel returned by
// BLiveClient.Events(). Inspect Type to determine which pointer field is set.
type Event struct {
	Type            string
	Danmaku         *DanmakuInfo
	Gift            *GiftData
	UserToastV2     *UserToastV2Data
	SuperChat       *SuperChatData
	SuperChatDelete *SuperChatDeleteData
}
