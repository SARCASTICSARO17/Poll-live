package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	sendBufferSize = 256
)

// Client represents a single connected browser watching a poll.
type Client struct {
	hub      *WebSocketHub
	pollID   string
	conn     *websocket.Conn
	send     chan []byte
	closeOnce sync.Once
}

// WebSocketHub tracks live connections per poll, manages on-demand Redis
// subscriptions and broadcasts updates only to the clients of the affected poll.
type WebSocketHub struct {
	redis *RedisService

	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}

	subsMu sync.Mutex
	subs   map[string]*redisSubscription
}

type redisSubscription struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// NewWebSocketHub creates an empty hub.
func NewWebSocketHub(redis *RedisService) *WebSocketHub {
	return &WebSocketHub{
		redis:   redis,
		clients: make(map[string]map[*Client]struct{}),
		subs:    make(map[string]*redisSubscription),
	}
}

// Register adds a client to the hub and lazily starts the Redis subscriber for
// its poll when it is the first viewer.
func (h *WebSocketHub) Register(c *Client) {
	h.mu.Lock()
	if h.clients[c.pollID] == nil {
		h.clients[c.pollID] = make(map[*Client]struct{})
	}
	h.clients[c.pollID][c] = struct{}{}
	first := len(h.clients[c.pollID]) == 1
	h.mu.Unlock()

	if first {
		h.ensureSubscription(c.pollID)
	}
}

// Unregister removes a client. When the last viewer of a poll leaves, the
// poll's Redis subscription is torn down to avoid leaking goroutines.
func (h *WebSocketHub) Unregister(c *Client) {
	h.mu.Lock()
	if clients, ok := h.clients[c.pollID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.clients, c.pollID)
		}
	}
	last := h.clients[c.pollID] == nil
	h.mu.Unlock()

	if last {
		h.stopSubscription(c.pollID)
	}
}

// Broadcast sends a raw payload to every connected client of a poll. Slow or
// dead clients are disconnected rather than blocking the broadcaster.
func (h *WebSocketHub) Broadcast(pollID string, payload []byte) {
	h.mu.RLock()
	clients := h.clients[pollID]
	list := make([]*Client, 0, len(clients))
	for c := range clients {
		list = append(list, c)
	}
	h.mu.RUnlock()

	for _, c := range list {
		select {
		case c.send <- payload:
		default:
			c.close()
		}
	}
}

// ensureSubscription starts a Redis Pub/Sub subscriber for a poll if one is not
// already running. The subscriber bridges Redis events into hub broadcasts.
func (h *WebSocketHub) ensureSubscription(pollID string) {
	h.subsMu.Lock()
	defer h.subsMu.Unlock()

	if _, exists := h.subs[pollID]; exists {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	sub, err := h.redis.Subscribe(ctx, UpdateChannel(pollID))
	if err != nil {
		log.Printf("ws: failed to subscribe to %s: %v", UpdateChannel(pollID), err)
		cancel()
		return
	}

	done := make(chan struct{})
	h.subs[pollID] = &redisSubscription{cancel: cancel, done: done}

	go func() {
		defer close(done)
		defer sub.Close()
		ch := sub.Channel()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				h.Broadcast(pollID, []byte(msg.Payload))
			case <-ctx.Done():
				return
			}
		}
	}()
}

// stopSubscription cancels and removes the Redis subscriber for a poll.
func (h *WebSocketHub) stopSubscription(pollID string) {
	h.subsMu.Lock()
	sub := h.subs[pollID]
	delete(h.subs, pollID)
	h.subsMu.Unlock()

	if sub != nil {
		sub.cancel()
		select {
		case <-sub.done:
		case <-time.After(2 * time.Second):
		}
	}
}

// ClientCount reports the number of connected clients for a poll (used in tests).
func (h *WebSocketHub) ClientCount(pollID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[pollID])
}

// ServePoll manages the full lifecycle of a single WebSocket connection for a
// poll: it wires the reader/writer pumps and registers the client with the hub.
func (h *WebSocketHub) ServePoll(conn *websocket.Conn, pollID string) {
	client := &Client{
		hub:    h,
		pollID: pollID,
		conn:   conn,
		send:   make(chan []byte, sendBufferSize),
	}

	h.Register(client)
	defer h.Unregister(client)

	go client.writePump()
	client.readPump()
}

const closeMessage = "\n"

func (c *Client) readPump() {
	defer c.close()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte(closeMessage))
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) close() {
	c.closeOnce.Do(func() {
		_ = c.conn.Close()
	})
}