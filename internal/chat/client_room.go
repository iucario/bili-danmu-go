package chat

import (
	"log/slog"
	"sync"
)

const subscriberBufSize = 64

type subscriber struct {
	ch   chan []byte
	done <-chan struct{} // request context Done channel
}

// ClientRoom fan-outs pre-formatted SSE bytes to all active subscribers.
type ClientRoom struct {
	mu   sync.Mutex
	subs []*subscriber
}

func NewClientRoom() *ClientRoom {
	return &ClientRoom{}
}

// Subscribe registers a new subscriber. The returned channel receives SSE frames.
// The caller must call the returned unsubscribe func when done.
func (r *ClientRoom) Subscribe(done <-chan struct{}) (<-chan []byte, func()) {
	sub := &subscriber{
		ch:   make(chan []byte, subscriberBufSize),
		done: done,
	}
	r.mu.Lock()
	r.subs = append(r.subs, sub)
	r.mu.Unlock()

	unsub := func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		for i, s := range r.subs {
			if s == sub {
				r.subs = append(r.subs[:i], r.subs[i+1:]...)
				break
			}
		}
	}
	return sub.ch, unsub
}

// Broadcast sends data to all subscribers. Slow subscribers whose buffers are
// full are dropped with a warning.
func (r *ClientRoom) Broadcast(data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()

	live := r.subs[:0]
	for _, s := range r.subs {
		select {
		case <-s.done:
			// Context cancelled; drop silently.
			continue
		default:
		}

		select {
		case s.ch <- data:
			live = append(live, s)
		default:
			slog.Warn("chat: dropping slow subscriber")
		}
	}
	r.subs = live
}

// Len returns the current subscriber count.
func (r *ClientRoom) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.subs)
}
