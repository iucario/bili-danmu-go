package bili

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ErrTooManyRetries is returned after 10 consecutive reconnect failures.
var ErrTooManyRetries = errors.New("too many retries connecting to Bilibili")

// ErrRoomNotFound is returned when Bilibili reports the room does not exist.
var ErrRoomNotFound = errors.New("room not found")

// biliUserAgent must be consistent across all API calls and the WS handshake
// because Bilibili binds the danmu token to the User-Agent it was issued for.
const biliUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36" +
	" (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

const maxRetries = 10

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

type hostEntry struct {
	Host    string
	WssPort int
}

// BLiveClient connects to a Bilibili live room and delivers decoded events on
// the channel returned by Events(). Call Start to begin the connection loop and
// Stop to shut it down cleanly.
type BLiveClient struct {
	roomID int64
	chanH  *chanHandler

	// OnConnect is called once per successful connection after the real room ID
	// and owner UID are resolved. May be called again on reconnect.
	OnConnect func(realRoomID, ownerUID int64)

	hc     *http.Client
	wbi    *wbiSigner
	jar    http.CookieJar
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Option configures a BLiveClient.
type Option func(*BLiveClient)

// WithBufferSize sets the capacity of the Events() channel (default: 64).
// Increase this for high-traffic rooms to avoid dropped events.
func WithBufferSize(n int) Option {
	return func(c *BLiveClient) {
		c.chanH = newChanHandlerSized(n)
	}
}

// NewBLiveClient creates a client for the given room ID.
// Call Events to receive the decoded live-room event stream.
func NewBLiveClient(roomID int64, opts ...Option) Client {
	ctx, cancel := context.WithCancel(context.Background())
	h := newChanHandler()
	jar, hc, wbi := getShared()
	c := &BLiveClient{
		roomID: roomID,
		chanH:  h,
		hc:     hc,
		wbi:    wbi,
		jar:    jar,
		ctx:    ctx,
		cancel: cancel,
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Events returns a read-only channel of live-room events. The channel is
// buffered (default size 64, configurable via WithBufferSize) and closed when
// the client stops — either because Stop was called or because a fatal
// connection error occurred. Call Err after the channel closes to distinguish
// the two cases.
func (c *BLiveClient) Events() <-chan Event {
	return c.chanH.ch
}

// Err returns the fatal error that caused the event channel to close, or nil
// if the client was stopped cleanly via Stop. Only meaningful after Events()
// has been closed.
func (c *BLiveClient) Err() error {
	p := c.chanH.fatalErr.Load()
	if p == nil {
		return nil
	}
	return *p
}

// SetOnConnect registers a callback invoked after each successful connection.
func (c *BLiveClient) SetOnConnect(fn func(realRoomID, ownerUID int64)) {
	c.OnConnect = fn
}

// Start begins the connection loop in a background goroutine.
func (c *BLiveClient) Start() {
	c.wg.Add(1)
	go c.runLoop()
}

// Stop signals the client to disconnect and waits for cleanup.
func (c *BLiveClient) Stop() {
	c.cancel()
	c.wg.Wait()
}

// runLoop implements the reconnect policy.
func (c *BLiveClient) runLoop() {
	defer c.wg.Done()
	defer c.chanH.close()

	totalRetries := 0
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		err := c.connect(c.ctx)
		if err == nil {
			return
		}
		if errors.Is(err, ErrRoomNotFound) {
			c.chanH.OnFatalError(ErrRoomNotFound)
			return
		}

		totalRetries++
		if totalRetries >= maxRetries {
			c.chanH.OnFatalError(ErrTooManyRetries)
			return
		}

		interval := time.Duration(min(1+(totalRetries-1)*2, 20))*time.Second +
			time.Duration(rand.Intn(3000))*time.Millisecond

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

// connect performs the full init sequence and runs the read loop until disconnect.
func (c *BLiveClient) connect(ctx context.Context) error {
	if err := c.fetchBuvid(ctx); err != nil {
		_ = err // non-fatal, continue without buvid
	}

	realRoomID, ownerUID, err := c.getRoomInfo(ctx)
	if err != nil {
		return fmt.Errorf("getRoomInfo: %w", err)
	}
	if c.OnConnect != nil {
		c.OnConnect(realRoomID, ownerUID)
	}

	hosts, token, err := c.getDanmuInfo(ctx, realRoomID)
	if err != nil {
		return fmt.Errorf("getDanmuInfo: %w", err)
	}
	if len(hosts) == 0 {
		return fmt.Errorf("no danmu host returned")
	}

	wsURL := fmt.Sprintf("wss://%s:%d/sub", hosts[0].Host, hosts[0].WssPort)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, http.Header{"User-Agent": {biliUserAgent}})
	if err != nil {
		return fmt.Errorf("wss dial %s: %w", wsURL, err)
	}
	defer func() { _ = conn.Close() }()

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
	if err := c.waitAuthReply(conn); err != nil {
		return fmt.Errorf("auth reply: %w", err)
	}

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
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := conn.WriteMessage(websocket.BinaryMessage, hbFrame); err != nil {
					return
				}
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
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
			continue
		}
		for _, f := range frames {
			if f.Header.Operation == OpSendMsgReply {
				c.chanH.dispatch(f.Body)
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

// ---- HTTP helpers ----

func (c *BLiveClient) fetchBuvid(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://www.bilibili.com/", nil)
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

func (c *BLiveClient) getRoomInfo(ctx context.Context) (roomID, uid int64, err error) {
	endpoint := fmt.Sprintf(
		"https://api.live.bilibili.com/room/v1/Room/get_info?room_id=%d", c.roomID)
	req, _ := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
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
	if len(result.Data) == 0 || result.Data[0] == '[' {
		return 0, 0, ErrRoomNotFound
	}
	var data roomInfoData
	if err := json.Unmarshal(result.Data, &data); err != nil {
		return 0, 0, fmt.Errorf("getRoomInfo data decode: %w", err)
	}
	return data.RoomID, data.UID, nil
}

func (c *BLiveClient) getDanmuInfo(ctx context.Context, realRoomID int64) ([]hostEntry, string, error) {
	signed, err := c.wbi.Sign(map[string]string{
		"id":   fmt.Sprintf("%d", realRoomID),
		"type": "0",
	})
	if err != nil {
		return nil, "", err
	}
	endpoint := "https://api.live.bilibili.com/xlive/web-room/v1/index/getDanmuInfo?" + signed
	req, _ := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
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
