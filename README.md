# danmu-go

A lightweight server that receives danmaku (弹幕) from Bilibili live rooms and displays them as a transparent chat overlay in OBS.

Port of the relay layer from [blivechat](https://github.com/xfgryujk/blivechat).

---

## Quick Start (OBS streamer)

### 1. Download

Grab the latest binary for your OS from the [Releases](../../releases) page:

| OS | File |
|---|---|
| Windows | `danmu-go-windows-amd64.exe` |
| macOS (Apple Silicon) | `danmu-go-darwin-arm64` |
| macOS (Intel) | `danmu-go-darwin-amd64` |
| Linux | `danmu-go-linux-amd64` |

### 2. Run the server

**Windows:** Double-click the `.exe`. A console window opens and shows:
```
  danmu-go running → http://127.0.0.1:12450
```

**macOS:** The binary is not code-signed, so Gatekeeper will block it on first run. Open a terminal and run:
```sh
chmod +x danmu-go-darwin-arm64        # make it executable (first time only)
xattr -d com.apple.quarantine danmu-go-darwin-arm64  # allow it to run
./danmu-go-darwin-arm64
```

Keep the window open while streaming.

### 3. Find your Bilibili room ID

Open your live room in a browser. The room ID is the number in the URL:

```
https://live.bilibili.com/12345   →   room ID is 12345
```

### 4. Add to OBS

1. In OBS → **Sources** → **+** → **Browser**.
2. Set the URL to:
   ```
   http://127.0.0.1:12450/obs/?roomId=12345
   ```
   (replace `12345` with your room ID)
3. Set width/height to match your overlay area (e.g. 400 × 800).
4. Check **"Shutdown source when not visible"** to pause when the scene is inactive.
5. Click **OK**. The chat overlay appears with a transparent background.

> **Preview without OBS:** paste the URL directly into any browser to see the overlay on a checkerboard background.

---

## OBS overlay options

Append options to the URL as needed:

| Parameter | Values | Description |
|---|---|---|
| `roomId` | integer | **Required.** Bilibili live room ID. |
| `theme` | `plain` (default) · `bubble` · `bubble-light` | Visual theme. Bubble themes show user avatars. |
| `filterLottery` | `1` | Hide lottery/raffle danmaku. |

Example with all options:
```
http://127.0.0.1:12450/obs/?roomId=12345&theme=bubble&filterLottery=1
```

### Layout

- **Top zone** — Pinned Super Chat cards (≥¥30), colored by price tier, auto-removed after their paid duration.
- **Bottom zone** — Scrolling danmaku list, newest at the bottom. Badges for owner / admin / guard ranks. Medal name/level shown inline.

---

## Configuration (optional)

On first run, a `data/config.ini` file is created automatically. Open it with any text editor to change settings:

```ini
[server]
; Listening address.
;   127.0.0.1  — only this computer can connect (default, recommended)
;   0.0.0.0    — allow other devices on your local network
host = 127.0.0.1

; Port number. Change this if 12450 is already in use on your system.
port = 12450
```

Restart the server after editing.

---

## Troubleshooting

**The overlay shows "Add `?roomId=12345` to the URL"**  
→ The `roomId` parameter is missing from the OBS browser source URL.

**The overlay shows "Disconnected — reconnecting…"**  
→ The server isn't running, or the room ID is wrong. Check that `danmu-go` is still open in the terminal.

**Port 12450 is already in use**  
→ Change the port in `data/config.ini` and update the OBS URL to match.

**Logs** — if something goes wrong, share the log file:

| OS | Path |
|---|---|
| macOS / Linux | `/tmp/danmu-go.log` |
| Windows | `%TEMP%\danmu-go.log` |

---

## SSE API (for developers)

danmu-go exposes a simple HTTP API for building custom overlays or integrations.

### Connect to a room

```sh
curl -N 'http://127.0.0.1:12450/api/chat/stream?roomId=<ROOM_ID>'
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
const es = new EventSource(`http://127.0.0.1:12450/api/chat/stream?roomId=${roomId}`)
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

## Development

### Prerequisites

- Go 1.21+

### Project layout

```
danmu-go/
├── main.go                        # Entry point — wires config, room manager, HTTP server
├── config/config.go               # Config struct and defaults (host, port)
├── server/server.go               # HTTP server with graceful shutdown
├── api/
│   ├── chat.go                    # GET /api/chat/stream SSE handler
│   └── avatar.go                  # GET /api/avatar?url=… avatar proxy (bypasses Bilibili CORS)
└── internal/
    ├── bili/
    │   ├── frame.go               # Binary frame encode/decode, zlib/brotli decompress
    │   ├── models.go              # Raw Bilibili message structs (DanmakuInfo, GiftData, …)
    │   ├── client.go              # BLiveClient: WBI sign, init sequence, WSS, heartbeat, reconnect
    │   └── handler.go             # HandlerInterface + BaseHandler dispatch
    └── chat/
        ├── models.go              # SSE event structs (AddTextEvent, AddGiftEvent, …)
        ├── client_room.go         # Per-room SSE fan-out to subscriber channels
        ├── room_manager.go        # Room lifecycle: start on first sub, teardown after 10 s idle
        └── msg_handler.go         # Translates Bilibili messages → SSE events
```

### Build

```sh
go build .
./danmu-go
```

### Build with embedded overlay

```sh
cd apps/obs-plugin && pnpm install && pnpm build
cd ../..
go build -tags obs .
```

> **Without `-tags obs`** the server serves `/obs/` from `apps/obs-plugin/build/` on disk — ideal for development.

### Develop the overlay

**Terminal 1 — Go backend:**
```sh
go run .
```

**Terminal 2 — Svelte watch build:**
```sh
cd apps/obs-plugin
pnpm build:watch
```

Open `http://127.0.0.1:12450/obs/?roomId=<ROOM_ID>` in a browser. After editing a Svelte file, Vite rebuilds in ~100 ms — just refresh the tab to see the change.

> `pnpm dev` starts a Vite HMR server on port 5173, but it can't reach the Go SSE API (different origin). Use `build:watch` + the Go server for end-to-end testing.

### Lint / format

```sh
gofumpt -w .
golangci-lint run ./...
```

### Adding a new SSE event type

1. Add a struct to `internal/chat/models.go`
2. Handle the source Bilibili command in `internal/chat/msg_handler.go`
3. Call `h.broadcast("event_name", ev)`

### Roadmap

- [ ] Plugin SSE + inject (`internal/plugin/`, `api/plugin.go`)
- [ ] REST helpers (`api/room_info.go`)
