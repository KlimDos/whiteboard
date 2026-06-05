package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/alimov/whiteboard/internal/hub"
	"github.com/alimov/whiteboard/internal/id"
	"github.com/alimov/whiteboard/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsMessage struct {
	Type   string            `json:"type"`
	Color  string            `json:"color,omitempty"`
	Width  float64           `json:"width,omitempty"`
	X0     float64           `json:"x0,omitempty"`
	Y0     float64           `json:"y0,omitempty"`
	X1     float64           `json:"x1,omitempty"`
	Y1     float64           `json:"y1,omitempty"`
	Strokes []storage.Stroke `json:"strokes,omitempty"`
}

func (h *Handler) WebSocket(c *gin.Context) {
	sessionID := c.Param("id")
	if !id.Valid(sessionID) {
		c.Status(http.StatusNotFound)
		return
	}

	ctx := c.Request.Context()
	exists, err := h.store.SessionExists(ctx, sessionID)
	if err != nil || !exists {
		c.Status(http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := &hub.Client{Send: make(chan []byte, 64)}
	h.hub.Register(sessionID, client)
	defer func() {
		h.hub.Unregister(sessionID, client)
		conn.Close()
	}()

	strokes, err := h.store.ListStrokes(ctx, sessionID)
	if err != nil {
		log.Printf("list strokes: %v", err)
		return
	}
	history, _ := json.Marshal(wsMessage{Type: "history", Strokes: strokes})
	if err := conn.WriteMessage(websocket.TextMessage, history); err != nil {
		return
	}

	go writePump(conn, client)
	readPump(ctx, conn, h, sessionID, client)
}

func writePump(conn *websocket.Conn, client *hub.Client) {
	for msg := range client.Send {
		conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func readPump(ctx context.Context, conn *websocket.Conn, h *Handler, sessionID string, client *hub.Client) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg wsMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "stroke":
			stroke := storage.Stroke{
				Color: msg.Color,
				Width: msg.Width,
				X0:    msg.X0,
				Y0:    msg.Y0,
				X1:    msg.X1,
				Y1:    msg.Y1,
			}
			if err := h.store.AddStroke(ctx, sessionID, stroke); err != nil {
				log.Printf("add stroke: %v", err)
				continue
			}
			h.hub.Broadcast(sessionID, data, client)
		case "clear":
			if err := h.store.ClearStrokes(ctx, sessionID); err != nil {
				log.Printf("clear strokes: %v", err)
				continue
			}
			clearMsg, _ := json.Marshal(wsMessage{Type: "clear"})
			h.hub.Broadcast(sessionID, clearMsg, nil)
		}
	}
}
