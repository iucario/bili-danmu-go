package bili

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
)

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
