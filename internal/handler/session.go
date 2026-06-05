package handler

import (
	"net/http"

	"github.com/alimov/whiteboard/internal/id"
	"github.com/gin-gonic/gin"
)

func (h *Handler) Index(c *gin.Context) {
	h.render(c, "index.html", nil)
}

func (h *Handler) CreateSession(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID, err := id.Generate()
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create session")
		return
	}
	if err := h.store.CreateSession(ctx, sessionID); err != nil {
		c.String(http.StatusInternalServerError, "failed to create session")
		return
	}
	c.Redirect(http.StatusFound, "/"+sessionID)
}

func (h *Handler) JoinSession(c *gin.Context) {
	ctx := c.Request.Context()
	sessionID := c.PostForm("id")
	if !id.Valid(sessionID) {
		c.String(http.StatusNotFound, "session not found")
		return
	}
	exists, err := h.store.SessionExists(ctx, sessionID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to join session")
		return
	}
	if !exists {
		c.String(http.StatusNotFound, "session not found")
		return
	}
	c.Redirect(http.StatusFound, "/"+sessionID)
}
