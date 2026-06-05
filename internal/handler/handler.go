package handler

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/alimov/whiteboard/internal/hub"
	"github.com/alimov/whiteboard/internal/storage"
	"github.com/gin-gonic/gin"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

var reservedPaths = map[string]struct{}{
	"create": {},
	"join":   {},
	"ws":     {},
	"static": {},
	"health": {},
}

type Handler struct {
	store     storage.Storage
	hub       *hub.Hub
	templates *template.Template
}

func New(store storage.Storage, h *hub.Hub) (*Handler, error) {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Handler{store: store, hub: h, templates: tmpl}, nil
}

func RegisterRoutes(r *gin.Engine, store storage.Storage, h *hub.Hub) error {
	handler, err := New(store, h)
	if err != nil {
		return err
	}

	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		return err
	}
	r.StaticFS("/static", http.FS(static))

	r.GET("/", handler.Index)
	r.POST("/create", handler.CreateSession)
	r.POST("/join", handler.JoinSession)
	r.GET("/ws/:id", handler.WebSocket)
	r.GET("/:id", handler.Board)

	return nil
}

func (h *Handler) render(c *gin.Context, name string, data any) {
	c.Status(http.StatusOK)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.String(http.StatusInternalServerError, "template error")
	}
}

func isReserved(path string) bool {
	_, ok := reservedPaths[path]
	return ok
}
