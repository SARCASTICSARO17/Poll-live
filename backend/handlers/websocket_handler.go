package handlers

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"poll-live/backend/services"
	"poll-live/backend/utils"
)

// WebSocketHandler upgrades HTTP connections to WebSocket.
type WebSocketHandler struct {
	hub         *services.WebSocketHub
	allowedOrigins []string
}

// NewWebSocketHandler wires the WebSocket handler. allowedOrigins controls
// which browser origins may open WebSocket connections.
func NewWebSocketHandler(hub *services.WebSocketHub, allowedOrigins []string) *WebSocketHandler {
	return &WebSocketHandler{hub: hub, allowedOrigins: allowedOrigins}
}

func (h *WebSocketHandler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // non-browser clients (tests, tools)
	}
	for _, allowed := range h.allowedOrigins {
		if originsMatch(origin, allowed) {
			return true
		}
	}
	return false
}

func originsMatch(origin, allowed string) bool {
	if origin == allowed {
		return true
	}
	u1, err1 := url.Parse(origin)
	u2, err2 := url.Parse(allowed)
	if err1 != nil || err2 != nil {
		return false
	}
	return strings.EqualFold(u1.Host, u2.Host) && u1.Scheme == u2.Scheme
}

// Serve handles GET /api/polls/:id/ws.
func (h *WebSocketHandler) Serve(c *gin.Context) {
	pollID, err := parsePollID(c)
	if err != nil {
		utils.Error(c, http.StatusBadRequest, "invalid poll id")
		return
	}

	up := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}

	conn, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("ws: upgrade failed: %v", err)
		return
	}

	h.hub.ServePoll(conn, pollID.Hex())
}