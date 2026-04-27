package bili

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// ErrTooManyRetries is returned after 30 consecutive reconnect failures.
var ErrTooManyRetries = errors.New("too many retries connecting to Bilibili")

// ErrRoomNotFound is returned when Bilibili reports the room does not exist.
var ErrRoomNotFound = errors.New("room not found")

// biliUserAgent must be consistent across all API calls and the WS handshake
// because Bilibili binds the danmu token to the User-Agent it was issued for.
const biliUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" +
	" (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

const maxRetries = 10

// sharedJar and sharedHTTPClient are shared across all BLiveClient instances so
// that the buvid3 cookie (and any login cookies) obtained by one room's init
// sequence are reused by all rooms, matching blivechat's single-session design.
var (
	sharedJarOnce sync.Once
	sharedJar     http.CookieJar
	sharedHC      *http.Client
	sharedWBI     *wbiSigner
	sharedUID     atomic.Int64 // logged-in viewer UID, 0 if anonymous
)

func getShared() (http.CookieJar, *http.Client, *wbiSigner) {
	sharedJarOnce.Do(func() {
		jar, _ := cookiejar.New(nil)
		sharedJar = jar
		sharedHC = &http.Client{Jar: jar, Timeout: 15 * time.Second}
		sharedWBI = newWbiSigner(sharedHC)
	})
	return sharedJar, sharedHC, sharedWBI
}

// wbiKeyIndexTable is the shuffle permutation used for WBI signing.
var wbiKeyIndexTable = []int{
	46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35,
	27, 43, 5, 49, 33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13,
}

// wbiFilterChars are characters stripped from WBI parameter values.
const wbiFilterChars = "!'()*"

// wbiSigner fetches and caches the WBI signing key.
type wbiSigner struct {
	mu        sync.Mutex
	mixedKey  string
	expiresAt time.Time
	client    *http.Client
}

func newWbiSigner(hc *http.Client) *wbiSigner {
	return &wbiSigner{client: hc}
}

func (w *wbiSigner) getKey() (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if time.Now().Before(w.expiresAt) && w.mixedKey != "" {
		return w.mixedKey, nil
	}

	req, _ := http.NewRequest("GET", "https://api.bilibili.com/x/web-interface/nav", nil)
	setBiliHeaders(req)
	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("wbi nav fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Data struct {
			WbiImg struct {
				ImgURL string `json:"img_url"`
				SubURL string `json:"sub_url"`
			} `json:"wbi_img"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("wbi nav decode: %w", err)
	}

	imgKey := stemFromURL(result.Data.WbiImg.ImgURL)
	subKey := stemFromURL(result.Data.WbiImg.SubURL)
	shuffled := imgKey + subKey

	key := make([]byte, 0, 32)
	for _, i := range wbiKeyIndexTable {
		if i < len(shuffled) {
			key = append(key, shuffled[i])
		}
	}
	w.mixedKey = string(key)
	w.expiresAt = time.Now().Add(12 * time.Hour)
	return w.mixedKey, nil
}

// Sign adds wts and w_rid to params, returning a signed query string.
func (w *wbiSigner) Sign(params map[string]string) (string, error) {
	key, err := w.getKey()
	if err != nil {
		return "", err
	}

	params["wts"] = fmt.Sprintf("%d", time.Now().Unix())

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		v := filterWbiChars(params[k])
		parts = append(parts, url.QueryEscape(k)+"="+url.QueryEscape(v))
	}
	query := strings.Join(parts, "&")

	hash := md5.Sum([]byte(query + key))
	wRid := fmt.Sprintf("%x", hash)

	return query + "&w_rid=" + wRid, nil
}

func stemFromURL(rawURL string) string {
	p := path.Base(rawURL)
	if idx := strings.LastIndex(p, "."); idx != -1 {
		return p[:idx]
	}
	return p
}

func filterWbiChars(s string) string {
	return strings.Map(func(r rune) rune {
		if strings.ContainsRune(wbiFilterChars, r) {
			return -1
		}
		return r
	}, s)
}

// ---- Bilibili API responses ----

type roomInfoResp struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

type roomInfoData struct {
	RoomID int64 `json:"room_id"`
	UID    int64 `json:"uid"`
}

type danmuInfoResp struct {
	Code int `json:"code"`
	Data struct {
		Token    string `json:"token"`
		HostList []struct {
			Host    string `json:"host"`
			WssPort int    `json:"wss_port"`
		} `json:"host_list"`
	} `json:"data"`
}

// ---- BLiveClient ----

// BLiveClient connects to a Bilibili live room and dispatches messages.
type BLiveClient struct {
	roomID  int64 // user-supplied room ID (may be short ID)
	handler HandlerInterface

	// OnConnect is called once per successful connection, after the real
	// room ID and owner UID are resolved. May be called multiple times on reconnect.
	OnConnect func(realRoomID, ownerUID int64)

	hc     *http.Client
	wbi    *wbiSigner
	jar    http.CookieJar
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewBLiveClient creates a client for the given room ID.
// handler receives decoded business messages.
func NewBLiveClient(roomID int64, handler HandlerInterface) *BLiveClient {
	jar, hc, wbi := getShared()
	return &BLiveClient{
		roomID:  roomID,
		handler: handler,
		hc:      hc,
		wbi:     wbi,
		jar:     jar,
		stopCh:  make(chan struct{}),
	}
}

// SetSESSDATA updates the shared cookie jar used by all Bilibili API calls and
// future WebSocket auth frames. Existing live room connections must reconnect
// to pick up the new cookie and viewer UID.
func SetSESSDATA(sessdata string) {
	_, _, _ = getShared() // ensure jar is initialised
	// Set the cookie for every bilibili.com host we talk to.
	hosts := []string{
		"https://bilibili.com",
		"https://www.bilibili.com",
		"https://api.bilibili.com",
		"https://api.live.bilibili.com",
		"https://passport.bilibili.com",
	}
	ck := &http.Cookie{Name: "SESSDATA", Value: sessdata, Path: "/"}
	if sessdata == "" {
		ck.MaxAge = -1
		ck.Expires = time.Unix(0, 0)
		sharedUID.Store(0)
	}
	for _, h := range hosts {
		u, _ := url.Parse(h)
		sharedJar.SetCookies(u, []*http.Cookie{ck})
	}
	// Fetch viewer UID so we can send it in the WS auth frame.
	go func() {
		uid, err := getViewerUID(sharedHC)
		if err != nil {
			slog.Warn("bili: could not fetch viewer UID", "err", err)
			return
		}
		sharedUID.Store(uid)
		slog.Info("bili: SESSDATA loaded", "uid", uid)
	}()
}

// Start begins the connection loop in a background goroutine.
func (c *BLiveClient) Start() {
	c.wg.Add(1)
	go c.runLoop()
}

// Stop signals the client to disconnect and waits for cleanup.
func (c *BLiveClient) Stop() {
	close(c.stopCh)
	c.wg.Wait()
}

// runLoop implements the reconnect policy from blivechat.
func (c *BLiveClient) runLoop() {
	defer c.wg.Done()

	totalRetries := 0
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		err := c.connect()
		if err == nil {
			// Clean disconnect (stopCh was closed)
			return
		}
		if errors.Is(err, ErrRoomNotFound) {
			slog.Error("bili: room not found, stopping", "roomID", c.roomID)
			if h, ok := c.handler.(interface{ OnFatalError(err error) }); ok {
				h.OnFatalError(ErrRoomNotFound)
			}
			return
		}

		totalRetries++
		if totalRetries >= maxRetries {
			slog.Error("bili: too many retries", "roomID", c.roomID)
			// Signal fatal error upstream via a special handler call if supported.
			if h, ok := c.handler.(interface{ OnFatalError(err error) }); ok {
				h.OnFatalError(ErrTooManyRetries)
			}
			return
		}

		interval := time.Duration(min(1+(totalRetries-1)*2, 20))*time.Second +
			time.Duration(rand.Intn(3000))*time.Millisecond
		slog.Info("bili: reconnecting", "roomID", c.roomID, "retry", totalRetries, "in", interval, "err", err)

		select {
		case <-c.stopCh:
			return
		case <-time.After(interval):
		}
	}
}

// connect performs the full init sequence and runs until disconnect.
func (c *BLiveClient) connect() error {
	// Step 1: fetch buvid3 cookie from bilibili.com
	if err := c.fetchBuvid(); err != nil {
		slog.Warn("bili: fetchBuvid", "err", err)
		// Non-fatal: continue without buvid
	}

	// Step 2: resolve real room_id and owner uid
	realRoomID, ownerUID, err := c.getRoomInfo()
	if err != nil {
		return fmt.Errorf("getRoomInfo: %w", err)
	}

	if c.OnConnect != nil {
		c.OnConnect(realRoomID, ownerUID)
	}

	// Step 3: WBI-signed getDanmuInfo
	hosts, token, err := c.getDanmuInfo(realRoomID)
	if err != nil {
		return fmt.Errorf("getDanmuInfo: %w", err)
	}
	if len(hosts) == 0 {
		return fmt.Errorf("no danmu host returned")
	}

	// Step 4: WSS connect — User-Agent must match the one used to fetch the danmu token
	// (Bilibili signs the token with the UA).
	wsURL := fmt.Sprintf("wss://%s:%d/sub", hosts[0].Host, hosts[0].WssPort)
	wsHeader := http.Header{"User-Agent": {biliUserAgent}}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	conn, _, err := dialer.Dial(wsURL, wsHeader)
	if err != nil {
		return fmt.Errorf("wss dial %s: %w", wsURL, err)
	}
	defer func() { _ = conn.Close() }()

	// Step 4b: send AUTH frame.
	// Use the logged-in viewer UID when available; 0 = anonymous.
	buvid3 := c.getBuvid3()
	authBody, _ := json.Marshal(map[string]any{
		"uid":      sharedUID.Load(),
		"roomid":   realRoomID,
		"protover": 3,
		"platform": "web",
		"type":     2,
		"buvid":    buvid3,
		"key":      token,
	})
	if err := conn.WriteMessage(websocket.BinaryMessage, EncodeFrame(OpAuth, 1, authBody)); err != nil {
		return fmt.Errorf("write auth: %w", err)
	}

	// Wait for AUTH_REPLY
	if err := c.waitAuthReply(conn); err != nil {
		return fmt.Errorf("auth reply: %w", err)
	}
	slog.Info("room connected", "roomID", c.roomID)

	// Step 5: heartbeat goroutine — hbStop is closed when connect() returns,
	// stopping the goroutine regardless of whether it was a clean or error exit.
	hbStop := make(chan struct{})
	defer close(hbStop)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		hbFrame := EncodeFrame(OpHeartbeat, VerHeartbeat, []byte("{}"))
		for {
			select {
			case <-hbStop:
				return
			case <-c.stopCh:
				return
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.BinaryMessage, hbFrame); err != nil {
					return
				}
			}
		}
	}()

	dispatcher := &BaseHandler{Handler: c.handler}

	for {
		select {
		case <-c.stopCh:
			_ = conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return nil
		default:
		}

		if err := conn.SetReadDeadline(time.Now().Add(60 * time.Second)); err != nil {
			return fmt.Errorf("set read deadline: %w", err)
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		frames, err := DecodeFrames(data)
		if err != nil {
			slog.Warn("bili: decode frames", "err", err)
			continue
		}

		for _, f := range frames {
			switch f.Header.Operation {
			case OpSendMsgReply:
				dispatcher.Dispatch(f.Body)
			case OpHeartbeatReply:
				// first 4 bytes = popularity, ignore
			case OpAuthReply:
				// already handled
			}
		}
	}
}

// waitAuthReply reads the first frame and checks it is a successful AUTH_REPLY.
func (c *BLiveClient) waitAuthReply(conn *websocket.Conn) error {
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}
	_, data, err := conn.ReadMessage()
	if err != nil {
		return err
	}
	frames, err := DecodeFrames(data)
	if err != nil {
		return err
	}
	for _, f := range frames {
		if f.Header.Operation == OpAuthReply {
			var reply struct {
				Code int `json:"code"`
			}
			if err := json.Unmarshal(f.Body, &reply); err != nil {
				return fmt.Errorf("auth reply decode: %w", err)
			}
			if reply.Code != 0 {
				return fmt.Errorf("auth rejected, code=%d", reply.Code)
			}
			return nil
		}
	}
	return fmt.Errorf("no AUTH_REPLY received")
}

// ---- Init helpers ----

func (c *BLiveClient) fetchBuvid() error {
	req, _ := http.NewRequest("GET", "https://www.bilibili.com/", nil)
	setBiliHeaders(req)
	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return nil
}

func (c *BLiveClient) getBuvid3() string {
	u, _ := url.Parse("https://www.bilibili.com/")
	for _, ck := range c.jar.Cookies(u) {
		if ck.Name == "buvid3" {
			return ck.Value
		}
	}
	return ""
}

func (c *BLiveClient) getRoomInfo() (roomID, uid int64, err error) {
	endpoint := fmt.Sprintf(
		"https://api.live.bilibili.com/room/v1/Room/get_info?room_id=%d", c.roomID)
	req, _ := http.NewRequest("GET", endpoint, nil)
	setBiliHeaders(req)
	resp, err := c.hc.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	var result roomInfoResp
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, 0, err
	}
	if result.Code != 0 {
		return 0, 0, fmt.Errorf("get_info code=%d", result.Code)
	}
	// Bilibili returns data as [] (empty array) when the room doesn't exist.
	if len(result.Data) == 0 || result.Data[0] == '[' {
		return 0, 0, ErrRoomNotFound
	}
	var data roomInfoData
	if err := json.Unmarshal(result.Data, &data); err != nil {
		return 0, 0, fmt.Errorf("getRoomInfo data decode: %w", err)
	}
	return data.RoomID, data.UID, nil
}

type hostEntry struct {
	Host    string
	WssPort int
}

func (c *BLiveClient) getDanmuInfo(realRoomID int64) ([]hostEntry, string, error) {
	signed, err := c.wbi.Sign(map[string]string{
		"id":   fmt.Sprintf("%d", realRoomID),
		"type": "0",
	})
	if err != nil {
		return nil, "", err
	}
	endpoint := "https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuInfo?" + signed
	req, _ := http.NewRequest("GET", endpoint, nil)
	setBiliHeaders(req)
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	var result danmuInfoResp
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, "", err
	}
	if result.Code != 0 {
		return nil, "", fmt.Errorf("getDanmuInfo code=%d", result.Code)
	}

	hosts := make([]hostEntry, len(result.Data.HostList))
	for i, h := range result.Data.HostList {
		hosts[i] = hostEntry{Host: h.Host, WssPort: h.WssPort}
	}
	return hosts, result.Data.Token, nil
}

// getViewerUID fetches the logged-in user's UID from the Bilibili nav API.
func getViewerUID(hc *http.Client) (int64, error) {
	req, _ := http.NewRequest("GET", "https://api.bilibili.com/x/web-interface/nav", nil)
	setBiliHeaders(req)
	resp, err := hc.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	var result struct {
		Data struct {
			Mid int64 `json:"mid"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.Data.Mid, nil
}

func setBiliHeaders(req *http.Request) {
	req.Header.Set("User-Agent", biliUserAgent)
	req.Header.Set("Referer", "https://www.bilibili.com/")
	req.Header.Set("Origin", "https://www.bilibili.com")
}
