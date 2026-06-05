package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alimov/whiteboard/internal/hub"
	"github.com/alimov/whiteboard/internal/storage"
	"github.com/gin-gonic/gin"
)

func setupTestRouter(t *testing.T) (*gin.Engine, storage.Storage) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := storage.NewSQLite(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	r := gin.New()
	if err := RegisterRoutes(r, store, hub.New()); err != nil {
		t.Fatal(err)
	}
	return r, store
}

func TestCreateSession(t *testing.T) {
	r, store := setupTestRouter(t)
	ctx := context.Background()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/create", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("want 302, got %d", w.Code)
	}
	location := w.Header().Get("Location")
	if !strings.HasPrefix(location, "/") || len(location) != 9 {
		t.Fatalf("unexpected location: %q", location)
	}
	sessionID := strings.TrimPrefix(location, "/")
	exists, err := store.SessionExists(ctx, sessionID)
	if err != nil || !exists {
		t.Fatalf("session not created: exists=%v err=%v", exists, err)
	}
}

func TestJoinMissingSession(t *testing.T) {
	r, _ := setupTestRouter(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/join", strings.NewReader("id=aaaaaaaa"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
}

func TestWebSocketHistoryAndStroke(t *testing.T) {
	r, store := setupTestRouter(t)
	ctx := context.Background()

	const sessionID = "a3Kf9xQp"
	if err := store.CreateSession(ctx, sessionID); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/" + sessionID
	conn, err := websocketDial(wsURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var history wsMessage
	if err := conn.ReadJSON(&history); err != nil {
		t.Fatal(err)
	}
	if history.Type != "history" {
		t.Fatalf("want history, got %q", history.Type)
	}

	stroke := wsMessage{
		Type:  "stroke",
		Color: "#000",
		Width: 2,
		X0:    1, Y0: 2, X1: 3, Y1: 4,
	}
	if err := conn.WriteJSON(stroke); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(time.Second)
	for {
		strokes, err := store.ListStrokes(ctx, sessionID)
		if err != nil {
			t.Fatal(err)
		}
		if len(strokes) == 1 {
			if strokes[0].X1 != 3 {
				t.Fatalf("unexpected strokes: %+v", strokes)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("unexpected strokes: %+v", strokes)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
