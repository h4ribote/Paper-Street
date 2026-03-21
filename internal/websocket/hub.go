package websocket

import "sync"

type Hub struct {
	mu        sync.RWMutex
	clients   map[string]*Client
	broadcast chan Message
}

type Message struct {
	Topic string      `json:"topic"`
	Data  interface{} `json:"data"`
	TS    int64       `json:"ts"`
}

func NewHub() *Hub {
	return &Hub{
		clients:   make(map[string]*Client),
		broadcast: make(chan Message, 256),
	}
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client.ID] = client
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, client.ID)
}

func (h *Hub) Broadcast(message Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, client := range h.clients {
		client.Send(message)
	}
}
