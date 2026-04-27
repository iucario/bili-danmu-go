# bili-danmu-go

A lightweight server that receives danmaku (弹幕) from Bilibili live rooms and streams them as a transparent OBS chat overlay.

Port of the relay layer from [xfgryujk/blivechat](github.com/xfgryujk/blivechat).

---

## Quick Start

### 1. Download

Grab the latest binary from the [Releases](../../releases) page.

| OS | File |
|---|---|
| Windows | `bili-danmu-windows-amd64.exe` |
| macOS (Apple Silicon) | `bili-danmu-darwin-arm64` |
| macOS (Intel) | `bili-danmu-darwin-amd64` |
| Linux | `bili-danmu-linux-amd64` |

### 2. Run

**Windows:** Double-click the `.exe`.

**macOS:** Strip the quarantine flag first:

```sh
chmod +x bili-danmu-darwin-arm64
xattr -d com.apple.quarantine bili-danmu-darwin-arm64
./bili-danmu-darwin-arm64
```

The server starts at `http://127.0.0.1:5090`.

### 3. Add to OBS

1. **Sources → + → Browser**
2. URL: `http://127.0.0.1:5090/obs/?roomId=12345` (replace with your room ID)
3. Size: match your overlay area (e.g. 400 × 800)
4. Check **"Shutdown source when not visible"**

> Preview in a browser: paste the same URL into any browser tab.

---

## Configuration

On first run, a config file is created automatically:

| OS | Path |
|---|---|
| Windows | `%APPDATA%\bili-danmu-go\config.ini` |
| macOS | `~/Library/Application Support/bili-danmu-go/config.ini` |
| Linux | `~/.config/bili-danmu-go/config.ini` |

```ini
[server]
host = 127.0.0.1   ; use 0.0.0.0 to allow other devices on the LAN
port = 5090

[log]
level = info       ; debug, info, warn, error

[bilibili]
; sessdata =
```

Override `sessdata` with the `SESSDATA` environment variable. Restart the server after editing.

---

## Development

### Prerequisites

- Go 1.21+, pnpm

### Run locally (without embedding)

```sh
# Terminal 1 — Go backend
go run .

# Terminal 2 — Svelte watch build
cd apps/obs-plugin && pnpm install && pnpm build:watch
```

Open `http://127.0.0.1:5090/obs/?roomId=<ROOM_ID>` in a browser and refresh after Svelte changes.

### Build a self-contained binary

```sh
# Using Task (recommended)
task build

# Manual
cd apps/obs-plugin && pnpm install && pnpm build && cd ../..
cd apps/admin-web  && pnpm install && pnpm build && cd ../..
go build -tags obs -o bili-danmu-$(go env GOOS)-$(go env GOARCH) .
```

### Lint

```sh
gofumpt -w .
golangci-lint run ./...
```

### Project layout

```
├── main.go                   # Entry point
├── config/config.go          # Config (host, port, sessdata)
├── server/server.go          # HTTP server with graceful shutdown
├── api/                      # HTTP handlers (SSE, avatar proxy, config)
└── internal/
    ├── bili/                 # Bilibili WS client (WBI signing, frame codec, reconnect)
    └── chat/                 # Room manager, SSE fan-out, message translation
```

---

## SSE API (for developers)

danmu-go exposes a simple HTTP API for building custom overlays or integrations.

### Connect to a room

```sh
curl -N 'http://127.0.0.1:5090/api/chat/stream?roomId=<ROOM_ID>'
```

The server connects to Bilibili on first subscriber and tears down 10 s after the last subscriber disconnects.

### SSE event types

Each SSE event has a named type and a JSON payload.

| Event | When |
|---|---|
| `add_text` | New danmaku message |
| `add_gift` | Gift sent |
| `add_member` | Guard purchased |
| `add_super_chat` | Super Chat message |
| `del_super_chat` | Super Chat deleted |
| `fatal_error` | Connection unrecoverable (too many retries) |

**Example `add_text` payload:**

```
event: add_text
data: {"id":"...","timestamp":1713800000,"authorName":"xfgryujk",
       "authorType":0,"content":"hello","privilegeType":0,"isGiftDanmaku":false,
       "isLottery":false,"authorLevel":30,"isNewbie":false,"isMobileVerified":true,
       "medalLevel":12,"medalName":"守护","avatarUrl":"https://...","uid":"123456",
       "contentType":0,"contentTypeParams":{},"isMirror":false,"translation":""}
```

`authorType`: 0 normal · 1 guard · 2 admin · 3 room owner  
`privilegeType`: 0 none · 1 总督 · 2 提督 · 3 舰长  
`contentType`: 0 text · 1 emoticon (URL in `contentTypeParams.url`)  
`isLottery`: true when the danmaku is a lottery/raffle entry (`not_show` flag in Bilibili's wire protocol)

### Browser / TypeScript

```ts
const es = new EventSource(`http://127.0.0.1:5090/api/chat/stream?roomId=${roomId}`)
es.addEventListener('add_text', (e) => {
  const msg = JSON.parse(e.data)
  console.log(msg.authorName, msg.content)
})
es.addEventListener('add_gift', (e) => { /* ... */ })
es.onerror = () => console.error('SSE disconnected')
```

### Avatar proxy

Avatar images from Bilibili's CDN are blocked by browsers due to CORS. The server exposes `/api/avatar?url=<encoded-url>` to fetch images server-side. The OBS overlay rewrites all `avatarUrl` values through this proxy automatically.

---

### Roadmap

- [ ] Plugin SSE + inject (`internal/plugin/`, `api/plugin.go`)
- [ ] REST helpers (`api/room_info.go`)
