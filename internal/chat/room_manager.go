package chat

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/iucario/bili-danmu-go/pkg/bili"
)

const teardownDelay = 10 * time.Second

type managedRoom struct {
	room    *ClientRoom
	client  *bili.BLiveClient
	handler *LiveMsgHandler
	timer   *time.Timer
	nSubs   int
}

// RoomManager creates and tears down Bilibili live connections on demand.
// A room is started when the first SSE subscriber connects and torn down
// teardownDelay after the last subscriber leaves.
type RoomManager struct {
	mu    sync.Mutex
	rooms map[int64]*managedRoom
}

func NewRoomManager() *RoomManager {
	return &RoomManager{rooms: make(map[int64]*managedRoom)}
}

// Subscribe attaches ctx to a room, starting it if necessary.
// Returns a channel of pre-formatted SSE frames and an unsubscribe func.
// The caller must call unsubscribe when the request context is done.
func (m *RoomManager) Subscribe(ctx context.Context, roomID int64) (<-chan []byte, func()) {
	m.mu.Lock()

	r, exists := m.rooms[roomID]
	if !exists {
		r = m.startRoom(roomID)
	} else if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
	r.nSubs++

	ch, unsub := r.room.Subscribe(ctx.Done())
	m.mu.Unlock()

	return ch, func() {
		unsub()
		m.mu.Lock()
		defer m.mu.Unlock()
		r2, ok := m.rooms[roomID]
		if !ok {
			return
		}
		r2.nSubs--
		if r2.nSubs <= 0 {
			r2.nSubs = 0
			r2.timer = time.AfterFunc(teardownDelay, func() {
				m.teardown(roomID)
			})
		}
	}
}

// ActiveRooms returns the IDs of all currently managed rooms.
func (m *RoomManager) ActiveRooms() []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids := make([]int64, 0, len(m.rooms))
	for id := range m.rooms {
		ids = append(ids, id)
	}
	return ids
}

// StopAll stops all active rooms; used during graceful shutdown.
func (m *RoomManager) StopAll() {
	m.mu.Lock()
	ids := make([]int64, 0, len(m.rooms))
	for id := range m.rooms {
		ids = append(ids, id)
	}
	m.mu.Unlock()

	for _, id := range ids {
		m.teardown(id)
	}
}

// startRoom must be called with m.mu held.
func (m *RoomManager) startRoom(roomID int64) *managedRoom {
	room := NewClientRoom()

	onFatal := func() { m.teardown(roomID) }
	handler := NewLiveMsgHandler(roomID, room, onFatal)

	client := bili.NewBLiveClient(roomID, handler)
	client.OnConnect = handler.OnConnect

	r := &managedRoom{
		room:    room,
		client:  client,
		handler: handler,
	}
	m.rooms[roomID] = r
	client.Start()
	slog.Info("room started", "roomID", roomID)
	return r
}

func (m *RoomManager) teardown(roomID int64) {
	m.mu.Lock()
	r, ok := m.rooms[roomID]
	if !ok {
		m.mu.Unlock()
		return
	}
	if r.timer != nil {
		r.timer.Stop()
	}
	delete(m.rooms, roomID)
	m.mu.Unlock()

	r.client.Stop()
	slog.Info("room stopped", "roomID", roomID)
}
