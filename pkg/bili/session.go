package bili

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// sharedJar and sharedHC are shared across all BLiveClient instances so that
// the buvid3 cookie (and any login cookies) obtained by one room's init
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

// SetSESSDATA updates the shared cookie jar used by all Bilibili API calls and
// future WebSocket auth frames. Existing live room connections must reconnect
// to pick up the new cookie and viewer UID.
func SetSESSDATA(sessdata string) {
	_, _, _ = getShared() // ensure jar is initialised
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

// setBiliHeaders sets the standard Bilibili request headers.
// biliUserAgent must be consistent across all API calls and the WS handshake
// because Bilibili binds the danmu token to the User-Agent it was issued for.
func setBiliHeaders(req *http.Request) {
	req.Header.Set("User-Agent", biliUserAgent)
	req.Header.Set("Referer", "https://www.bilibili.com/")
	req.Header.Set("Origin", "https://www.bilibili.com")
}
