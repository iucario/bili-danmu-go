# danmu-go

A Go server that receives danmaku (弹幕) from Bilibili live rooms and relays them to browser clients via **Server-Sent Events (SSE)**.

Port of the relay layer from [blivechat](https://github.com/xfgryujk/blivechat).

---

## Usage

### Run

```sh
go run .
# Server listens on http://127.0.0.1:12450
```

### Connect to a room

```sh
curl -N 'http://127.0.0.1:12450/api/chat/stream?roomId=<ROOM_ID>'
```

Replace `<ROOM_ID>` with the Bilibili live room ID (short or full). The server connects to Bilibili on first subscriber and tears down 10 s after the last subscriber disconnects.

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
       "authorLevel":30,"isNewbie":false,"isMobileVerified":true,
       "medalLevel":12,"medalName":"守护","avatarUrl":"https://...","uid":"123456",
       "contentType":0,"contentTypeParams":{},"isMirror":false,"translation":""}
```

`authorType`: 0 normal · 1 guard · 2 admin · 3 room owner  
`privilegeType`: 0 none · 1 总督 · 2 提督 · 3 舰长  
`contentType`: 0 text · 1 emoticon (URL in `contentTypeParams.url`)

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

---

## OBS Browser Source overlay

danmu-go ships a built-in chat overlay page you can add directly to OBS.

### Add to OBS

1. Start `danmu-go` (default port 12450).
2. In OBS → **Sources** → **+** → **Browser**.
3. Set the URL to:
   ```
   http://127.0.0.1:12450/obs/?roomId=<ROOM_ID>
   ```
4. Set width/height to match your overlay area (e.g. 400 × 800).
5. Check **"Shutdown source when not visible"** to pause when the scene is inactive.

The overlay has a transparent background — it layers cleanly over gameplay footage.

### Layout

- **Top zone** — Pinned Super Chat cards, colored by price tier (¥30→¥2000+), auto-removed after their paid duration.
- **Bottom zone** — Scrolling danmaku list, newest at the bottom. Badges for owner / admin / guard ranks. Medal name/level shown inline.

### Build the overlay from source

The compiled files are embedded in the Go binary at build time via `-tags obs`. To build a self-contained release binary:

```sh
cd apps/obs-plugin
pnpm install
pnpm build
# Then rebuild Go with the obs tag:
cd ../..
go build -tags obs .
```

> **Without `-tags obs`** the server still starts normally. `/obs/` is served live from `apps/obs-plugin/build/` on disk instead — ideal for development (see below).

### Develop the overlay

In dev mode the Go server reads `apps/obs-plugin/build/` straight from disk, so you can iterate on the frontend without restarting Go.

**Terminal 1 — Go backend:**
```sh
go run .
```

**Terminal 2 — Svelte watch build:**
```sh
cd apps/obs-plugin
pnpm build:watch
```

**Confirm it works** — open in a browser (no OBS needed):
```
http://127.0.0.1:12450/obs/?roomId=<ROOM_ID>
```

You should see the transparent overlay. If no `roomId` is given, a hint message is shown. After editing a Svelte file, Vite rebuilds in ~100 ms — just refresh the browser tab (or click **Refresh** in the OBS Browser Source panel) to see the change.

> `pnpm dev` starts a Vite HMR server on port 5173, but it can't reach the Go SSE API (different origin). Use `build:watch` + the Go server for end-to-end testing.

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
│   └── chat.go                    # GET /api/chat/stream SSE handler
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

### Lint / format

```sh
gofumpt -w .
golangci-lint run ./...
```

### Adding a new SSE event type

1. Add a struct to `internal/chat/models.go`
2. Handle the source Bilibili command in `internal/chat/msg_handler.go`
3. Call `h.broadcast("event_name", ev)`

### Configuration

On first run, `data/config.ini` is created automatically next to the binary. Open it with any text editor:

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

To use a different config file location:

```sh
./danmu-go -config /path/to/config.ini
```

### Logs

The server logs to both the terminal and the OS temp directory:

| OS | Path |
|---|---|
| macOS / Linux | `/tmp/danmu-go.log` |
| Windows | `%TEMP%\danmu-go.log` |

Logs are appended across restarts. Share this file when reporting issues.

### Roadmap

- [ ] Plugin SSE + inject (`internal/plugin/`, `api/plugin.go`)
- [ ] REST helpers (`api/room_info.go`)
