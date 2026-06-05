package hub

import (
	"sync"
)

type Client struct {
	Send chan []byte
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]struct{}
}

func New() *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) Register(sessionID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[sessionID] == nil {
		h.rooms[sessionID] = make(map[*Client]struct{})
	}
	h.rooms[sessionID][c] = struct{}{}
}

func (h *Hub) Unregister(sessionID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	room := h.rooms[sessionID]
	if room == nil {
		return
	}
	delete(room, c)
	close(c.Send)
	if len(room) == 0 {
		delete(h.rooms, sessionID)
	}
}

func (h *Hub) Broadcast(sessionID string, msg []byte, except *Client) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.rooms[sessionID] {
		if client == except {
			continue
		}
		select {
		case client.Send <- msg:
		default:
		}
	}
}

func (h *Hub) ClientCount(sessionID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[sessionID])
}
