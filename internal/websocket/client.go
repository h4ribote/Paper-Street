package websocket

import "sync"

type Client struct {
	ID   string
	mu   sync.Mutex
	sent []Message
}

func NewClient(id string) *Client {
	return &Client{ID: id}
}

func (c *Client) Send(message Message) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, message)
}

func (c *Client) Messages() []Message {
	c.mu.Lock()
	defer c.mu.Unlock()
	messages := make([]Message, len(c.sent))
	copy(messages, c.sent)
	return messages
}
