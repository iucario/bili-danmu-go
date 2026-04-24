package bili

import "encoding/json"

// RawMessage is the top-level envelope for SEND_MSG_REPLY business messages.
type RawMessage struct {
	Cmd  string          `json:"cmd"`
	Data json.RawMessage `json:"data"`
	Info json.RawMessage `json:"info"`
}

// jsonStringOrInt unmarshals a JSON value that may be either a quoted string or
// a bare number (Bilibili sends uid as both depending on the message type).
type jsonStringOrInt string

func (s *jsonStringOrInt) UnmarshalJSON(b []byte) error {
	// Try string first
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		*s = jsonStringOrInt(str)
		return nil
	}
	// Fall back to number — convert to string
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*s = jsonStringOrInt(n.String())
	return nil
}

func (s jsonStringOrInt) String() string { return string(s) }

// ---- Danmaku (DANMU_MSG) ----
// info is a heterogeneous JSON array; we decode positionally.

type DanmakuInfo struct {
	// info[0] sub-fields
	Mode            int             `json:"-"`
	FontSize        int             `json:"-"`
	Color           int             `json:"-"`
	Timestamp       int64           `json:"-"`
	UidCrc32        string          `json:"-"`
	MsgType         int             `json:"-"` // info[0][9]: 0=normal,1=emoticon
	DmType          int             `json:"-"` // info[0][12]
	EmoticonOptions json.RawMessage `json:"-"` // info[0][13]
	VoiceConfig     json.RawMessage `json:"-"` // info[0][14]
	ModeInfo        json.RawMessage `json:"-"` // info[0][15]

	// info[1]
	Msg string `json:"-"`

	// info[2] user fields
	UID          int64  `json:"-"`
	Uname        string `json:"-"`
	Admin        int    `json:"-"` // 1=admin
	Vip          int    `json:"-"`
	Svip         int    `json:"-"`
	URank        int    `json:"-"`
	MobileVerify int    `json:"-"` // 1=verified
	UnameColor   string `json:"-"`

	// info[3] medal fields
	MedalLevel  int    `json:"-"`
	MedalName   string `json:"-"`
	MedalRoomID int64  `json:"-"`

	// info[4] user level
	UserLevel int `json:"-"`

	// info[5] title
	OldTitle string `json:"-"`
	Title    string `json:"-"`

	// info[7] privilege type
	PrivilegeType int `json:"-"`

	// info[16] wealth level
	WealthLevel int `json:"-"`

	// Mirror flag (set by caller when cmd==DANMU_MSG_MIRROR)
	IsMirror bool `json:"-"`
}

// ParseDanmakuInfo decodes the raw info array from a DANMU_MSG frame.
func ParseDanmakuInfo(raw json.RawMessage) (*DanmakuInfo, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, err
	}

	d := &DanmakuInfo{}

	// info[0]: sub-array
	if len(arr) > 0 {
		var sub []json.RawMessage
		if err := json.Unmarshal(arr[0], &sub); err == nil {
			getInt := func(i int) int {
				if i < len(sub) {
					var v int
					_ = json.Unmarshal(sub[i], &v)
					return v
				}
				return 0
			}
			getStr := func(i int) string {
				if i < len(sub) {
					var v string
					_ = json.Unmarshal(sub[i], &v)
					return v
				}
				return ""
			}
			getRaw := func(i int) json.RawMessage {
				if i < len(sub) {
					return sub[i]
				}
				return nil
			}
			d.Mode = getInt(1)
			d.FontSize = getInt(2)
			d.Color = getInt(3)
			var ts int64
			if len(sub) > 4 {
				_ = json.Unmarshal(sub[4], &ts)
			}
			d.Timestamp = ts
			d.UidCrc32 = getStr(7)
			d.MsgType = getInt(9)
			d.DmType = getInt(12)
			d.EmoticonOptions = getRaw(13)
			d.VoiceConfig = getRaw(14)
			d.ModeInfo = getRaw(15)
		}
	}

	// info[1]: message text
	if len(arr) > 1 {
		_ = json.Unmarshal(arr[1], &d.Msg)
	}

	// info[2]: user array
	if len(arr) > 2 {
		var user []json.RawMessage
		if err := json.Unmarshal(arr[2], &user); err == nil {
			getInt := func(i int) int {
				if i < len(user) {
					var v int
					_ = json.Unmarshal(user[i], &v)
					return v
				}
				return 0
			}
			getStr := func(i int) string {
				if i < len(user) {
					var v string
					_ = json.Unmarshal(user[i], &v)
					return v
				}
				return ""
			}
			var uid int64
			if len(user) > 0 {
				_ = json.Unmarshal(user[0], &uid)
			}
			d.UID = uid
			d.Uname = getStr(1)
			d.Admin = getInt(2)
			d.Vip = getInt(3)
			d.Svip = getInt(4)
			d.URank = getInt(5)
			d.MobileVerify = getInt(6)
			d.UnameColor = getStr(7)
		}
	}

	// info[3]: medal array
	if len(arr) > 3 {
		var medal []json.RawMessage
		if err := json.Unmarshal(arr[3], &medal); err == nil {
			getInt := func(i int) int {
				if i < len(medal) {
					var v int
					_ = json.Unmarshal(medal[i], &v)
					return v
				}
				return 0
			}
			getStr := func(i int) string {
				if i < len(medal) {
					var v string
					_ = json.Unmarshal(medal[i], &v)
					return v
				}
				return ""
			}
			d.MedalLevel = getInt(0)
			d.MedalName = getStr(1)
			var rid int64
			if len(medal) > 3 {
				_ = json.Unmarshal(medal[3], &rid)
			}
			d.MedalRoomID = rid
		}
	}

	// info[4]: user level array
	if len(arr) > 4 {
		var lvl []json.RawMessage
		if err := json.Unmarshal(arr[4], &lvl); err == nil && len(lvl) > 0 {
			_ = json.Unmarshal(lvl[0], &d.UserLevel)
		}
	}

	// info[5]: title array
	if len(arr) > 5 {
		var title []json.RawMessage
		if err := json.Unmarshal(arr[5], &title); err == nil {
			if len(title) > 0 {
				_ = json.Unmarshal(title[0], &d.OldTitle)
			}
			if len(title) > 1 {
				_ = json.Unmarshal(title[1], &d.Title)
			}
		}
	}

	// info[7]: privilege type
	if len(arr) > 7 {
		_ = json.Unmarshal(arr[7], &d.PrivilegeType)
	}

	// info[16]: wealth level array
	if len(arr) > 16 {
		var wealth []json.RawMessage
		if err := json.Unmarshal(arr[16], &wealth); err == nil && len(wealth) > 0 {
			_ = json.Unmarshal(wealth[0], &d.WealthLevel)
		}
	}

	return d, nil
}

// EmoticonOptionsData holds the emoticon URL when MsgType==1.
type EmoticonOptionsData struct {
	URL string `json:"url"`
}

// ModeInfoExtra carries reply info embedded in info[0][15].
type ModeInfoExtra struct {
	ReplyUname string `json:"reply_uname"`
}

// ModeInfoData is the partial shape of info[0][15].
type ModeInfoData struct {
	Extra string `json:"extra"` // JSON-encoded ModeInfoExtra
}

// ---- Gift (SEND_GIFT) ----

type GiftData struct {
	GiftName   string `json:"giftName"`
	Num        int    `json:"num"`
	Uname      string `json:"uname"`
	Face       string `json:"face"`
	GuardLevel int    `json:"guard_level"`
	UID        int64  `json:"uid"`
	Timestamp  int64  `json:"timestamp"`
	GiftID     int    `json:"giftId"`
	GiftType   int    `json:"giftType"`
	GiftInfo   struct {
		ImgBasic string `json:"img_basic"`
	} `json:"gift_info"`
	Action    string `json:"action"`
	Price     int    `json:"price"`
	Rnd       string `json:"rnd"`
	CoinType  string `json:"coin_type"` // "gold" or "silver"
	TotalCoin int    `json:"total_coin"`
	Tid       string `json:"tid"`
	MedalInfo struct {
		MedalLevel   int    `json:"medal_level"`
		MedalName    string `json:"medal_name"`
		AnchorRoomID int64  `json:"anchor_roomid"`
		TargetID     int64  `json:"target_id"`
	} `json:"medal_info"`
}

// ---- UserToastV2 (USER_TOAST_MSG_V2) — guard buy ----

type UserToastV2Data struct {
	SenderUinfo struct {
		UID  int64 `json:"uid"`
		Base struct {
			Name string `json:"name"`
			Face string `json:"face"`
		} `json:"base"`
	} `json:"sender_uinfo"`
	GuardInfo struct {
		GuardLevel int   `json:"guard_level"`
		StartTime  int64 `json:"start_time"`
		EndTime    int64 `json:"end_time"`
	} `json:"guard_info"`
	PayInfo struct {
		Num   int    `json:"num"`
		Price int    `json:"price"`
		Unit  string `json:"unit"`
	} `json:"pay_info"`
	GiftInfo struct {
		GiftID int `json:"gift_id"`
	} `json:"gift_info"`
	Option struct {
		Source int `json:"source"`
	} `json:"option"`
}

// ---- SuperChat (SUPER_CHAT_MESSAGE) ----

type SuperChatData struct {
	Price        int    `json:"price"`
	Message      string `json:"message"`
	MessageTrans string `json:"message_trans"`
	StartTime    int64  `json:"start_time"`
	EndTime      int64  `json:"end_time"`
	Time         int    `json:"time"`
	ID           int64  `json:"id"`
	Gift         struct {
		GiftID   int    `json:"gift_id"`
		GiftName string `json:"gift_name"`
	} `json:"gift"`
	UID      jsonStringOrInt `json:"uid"`
	UserInfo struct {
		Uname      string `json:"uname"`
		Face       string `json:"face"`
		GuardLevel int    `json:"guard_level"`
		UserLevel  int    `json:"user_level"`
	} `json:"user_info"`
	BackgroundColor       string `json:"background_color"`
	BackgroundBottomColor string `json:"background_bottom_color"`
	MedalInfo             struct {
		MedalLevel   int    `json:"medal_level"`
		MedalName    string `json:"medal_name"`
		AnchorRoomID int64  `json:"anchor_roomid"`
		TargetID     int64  `json:"target_id"`
	} `json:"medal_info"`
}

// ---- SuperChatDelete (SUPER_CHAT_MESSAGE_DELETE) ----

type SuperChatDeleteData struct {
	IDs []int64 `json:"ids"`
}
