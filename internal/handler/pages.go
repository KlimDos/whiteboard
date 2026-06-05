package handler

import (
	"net/http"

	"github.com/alimov/whiteboard/internal/id"
	"github.com/gin-gonic/gin"
)

type boardData struct {
	SessionID string
}

func (h *Handler) Board(c *gin.Context) {
	sessionID := c.Param("id")
	if isReserved(sessionID) || !id.Valid(sessionID) {
		c.Status(http.StatusNotFound)
		return
	}

	exists, err := h.store.SessionExists(c.Request.Context(), sessionID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load session")
		return
	}
	if !exists {
		c.Status(http.StatusNotFound)
		return
	}

	h.render(c, "board.html", boardData{SessionID: sessionID})
}
