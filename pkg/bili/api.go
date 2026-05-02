package bili

// Client is the interface for a Bilibili live room client.
// The concrete implementation is *BLiveClient, created with NewBLiveClient.
type Client interface {
	// Start begins the connection loop in a background goroutine.
	Start()
	// Stop signals the client to disconnect and waits for cleanup.
	Stop()
	// Events returns a read-only channel of live-room events. The channel is closed when the client stops (cleanly or due to a fatal error).
	Events() <-chan Event
	// Err returns the fatal error that caused Events() to close, or nil for a clean stop. Only meaningful after the Events() channel has been closed.
	Err() error
	// RealRoomID returns the resolved real room ID after the first successful connection, or 0 if not yet connected.
	RealRoomID() int64
	// OwnerUID returns the room owner's UID after the first successful connection, or 0 if not yet connected.
	OwnerUID() int64
}
