package websocket

import (
	"errors"
	"sync"
)

var ErrHubClosed = errors.New("stream is no longer available")
var ErrViewerLimit = errors.New("viewer limit reached for this stream")
var ErrStreamStopped = errors.New("stream is stopped")

type Hub struct {
	mu         sync.Mutex
	clients    map[*Client]struct{}
	closed     bool
	accepting  bool
	maxClients int
}

func NewHub(maxClients int) *Hub {
	return &Hub{clients: make(map[*Client]struct{}), maxClients: maxClients, accepting: true}
}

func (h *Hub) Subscribe(c *Client) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return ErrHubClosed
	}
	if !h.accepting {
		return ErrStreamStopped
	}
	if len(h.clients) >= h.maxClients {
		return ErrViewerLimit
	}
	h.clients[c] = struct{}{}
	return nil
}

func (h *Hub) Unsubscribe(c *Client) { h.mu.Lock(); delete(h.clients, c); h.mu.Unlock(); c.Close() }
func (h *Hub) Viewers() int          { h.mu.Lock(); defer h.mu.Unlock(); return len(h.clients) }

func (h *Hub) Broadcast(data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	for c := range h.clients {
		select {
		case <-c.done:
			delete(h.clients, c)
		case c.send <- data:
		default:
			// Disconnect rather than drop arbitrary MPEG bytes and corrupt a decoder.
			delete(h.clients, c)
			c.Close()
		}
	}
}

func (h *Hub) DisconnectAll() { h.mu.Lock(); defer h.mu.Unlock(); h.disconnectLocked() }
func (h *Hub) Close()         { h.mu.Lock(); defer h.mu.Unlock(); h.closed = true; h.disconnectLocked() }

// Stop also rejects handshakes that passed the HTTP status check before source cleanup.
func (h *Hub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.accepting = false
	h.disconnectLocked()
}

func (h *Hub) Start() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.closed {
		h.accepting = true
	}
}

func (h *Hub) disconnectLocked() {
	for c := range h.clients {
		c.Close()
		delete(h.clients, c)
	}
}
