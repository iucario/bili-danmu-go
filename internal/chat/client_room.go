package chat

import (
	"log/slog"
	"sync"
)

const subscriberBufSize = 64

// broadcaster is a generic fan-out channel. T is the message type.
type broadcaster[T any] struct {
	mu   sync.Mutex
	subs []*chanSub[T]
}

type chanSub[T any] struct {
	ch   chan T
	done <-chan struct{}
}

func (b *broadcaster[T]) subscribe(done <-chan struct{}) (<-chan T, func()) {
	s := &chanSub[T]{
		ch:   make(chan T, subscriberBufSize),
		done: done,
	}
	b.mu.Lock()
	b.subs = append(b.subs, s)
	b.mu.Unlock()

	return s.ch, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		for i, x := range b.subs {
			if x == s {
				b.subs = append(b.subs[:i], b.subs[i+1:]...)
				break
			}
		}
	}
}

func (b *broadcaster[T]) broadcast(v T, warnTag string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	live := b.subs[:0]
	for _, s := range b.subs {
		select {
		case <-s.done:
			continue
		default:
		}
		select {
		case s.ch <- v:
			live = append(live, s)
		default:
			slog.Warn("chat: dropping slow subscriber", "tag", warnTag)
		}
	}
	b.subs = live
}

func (b *broadcaster[T]) len() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.subs)
}

// ClientRoom fan-outs pre-formatted SSE bytes and typed ChatEvents to all active subscribers.
type ClientRoom struct {
	sse    broadcaster[[]byte]
	events broadcaster[ChatEvent]
}

func NewClientRoom() *ClientRoom { return &ClientRoom{} }

// Subscribe registers an SSE-bytes subscriber. Call the returned func to unsubscribe.
func (r *ClientRoom) Subscribe(done <-chan struct{}) (<-chan []byte, func()) {
	return r.sse.subscribe(done)
}

// SubscribeEvents registers an in-process typed event subscriber. Call the returned func to unsubscribe.
func (r *ClientRoom) SubscribeEvents(done <-chan struct{}) (<-chan ChatEvent, func()) {
	return r.events.subscribe(done)
}

// Broadcast sends raw SSE bytes to all SSE subscribers.
func (r *ClientRoom) Broadcast(data []byte) {
	r.sse.broadcast(data, "sse")
}

// BroadcastEvent sends a typed ChatEvent to all in-process event subscribers.
func (r *ClientRoom) BroadcastEvent(ev ChatEvent) {
	r.events.broadcast(ev, "event")
}

// Len returns the current SSE subscriber count.
func (r *ClientRoom) Len() int { return r.sse.len() }
