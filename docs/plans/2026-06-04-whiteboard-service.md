# Whiteboard Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a minimal real-time whiteboard at `wb.alimov.top/{randomID}` with a landing page to create or join sessions, backed by persistent storage.

**Architecture:** Single Go binary (Gin) serves HTML pages, a REST endpoint to create sessions, and WebSocket connections for live stroke sync. SQLite stores session metadata and stroke events; an in-memory hub fans out WebSocket messages to clients in the same room. No auth — anyone with the URL can draw.

**Tech Stack:** Go 1.22+, Gin, gorilla/websocket, modernc.org/sqlite (pure Go, no CGO), embedded HTML templates + vanilla JS canvas.

---

## System Design

### Requirements

| Requirement | Decision |
|-------------|----------|
| Create session | Button on `/` → generates random ID → redirect to `/{id}` |
| Join session | Enter ID on `/` → redirect to `/{id}` (404 if missing) |
| Real-time sync | WebSocket per session room |
| Persistence | SQLite — strokes survive server restart |
| Auth | None |
| Scale | Single instance, ~1–5 concurrent users, personal use |

### URL Map

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/` | Landing page: Create / Join |
| POST | `/create` | Create session, redirect to `/{id}` |
| POST | `/join` | Validate ID exists, redirect to `/{id}` |
| GET | `/{id}` | Whiteboard page (8-char alphanumeric ID) |
| GET | `/ws/{id}` | WebSocket upgrade for live drawing |

Reserved paths (`create`, `join`, `ws`, `static`) must not be treated as session IDs.

### Data Model

```sql
CREATE TABLE sessions (
    id         TEXT PRIMARY KEY,          -- 8-char base62, e.g. "a3Kf9xQp"
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE strokes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    payload    TEXT NOT NULL,             -- JSON: {color, width, x0, y0, x1, y1}
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_strokes_session ON strokes(session_id, id);
```

**Session ID generation:** 8 characters from `[A-Za-z0-9]`, cryptographically random (`crypto/rand`). Retry on collision (negligible at this scale).

### WebSocket Protocol

All messages are JSON text frames.

**Client → Server**

```json
{"type": "stroke", "color": "#000000", "width": 3, "x0": 10, "y0": 20, "x1": 15, "y1": 25}
{"type": "clear"}
```

**Server → Client**

```json
{"type": "history", "strokes": [{"color":"#000","width":3,"x0":1,"y0":2,"x1":3,"y1":4}, ...]}
{"type": "stroke", "color": "#000000", "width": 3, "x0": 10, "y0": 20, "x1": 15, "y1": 25}
{"type": "clear"}
```

**Flow on connect:**
1. Client opens `ws://host/ws/{id}`
2. Server validates session exists
3. Server sends `history` with all stored strokes
4. Client draws incoming strokes on canvas
5. On local draw, client sends `stroke`; server persists + broadcasts to other clients in room
6. On clear, server deletes strokes for session + broadcasts `clear`

### Component Diagram

```
┌─────────────────────────────────────────────────────────┐
│                     Browser                              │
│  index.html (Create / Join)    board.html + canvas JS   │
└───────────────┬─────────────────────────┬───────────────┘
                │ HTTP                     │ WebSocket
                ▼                          ▼
┌───────────────────────────────────────────────────────────┐
│                    Gin Server (cmd/server)                 │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐  │
│  │ PageHandler │  │ SessionHandler│  │ WSHandler       │  │
│  └──────┬──────┘  └──────┬───────┘  └────────┬────────┘  │
│         │                │                    │            │
│         └────────────────┼────────────────────┘            │
│                          ▼                                 │
│                   ┌─────────────┐    ┌──────────────┐      │
│                   │   Storage   │    │   Hub (mem)  │      │
│                   │   (SQLite)  │    │  room→clients│      │
│                   └─────────────┘    └──────────────┘      │
└───────────────────────────────────────────────────────────┘
                          │
                          ▼
                   whiteboard.db (volume)
```

### Project Layout

```
whiteboard/
├── cmd/server/main.go
├── internal/
│   ├── id/id.go                 # random ID generation + validation
│   ├── storage/
│   │   ├── storage.go           # interface
│   │   └── sqlite.go            # SQLite implementation
│   ├── hub/hub.go               # WebSocket room fan-out
│   └── handler/
│       ├── pages.go             # GET /, GET /{id}
│       ├── session.go           # POST /create, POST /join
│       └── websocket.go         # GET /ws/{id}
├── web/
│   ├── templates/
│   │   ├── index.html
│   │   └── board.html
│   └── static/
│       ├── board.js
│       └── style.css
├── go.mod
├── go.sum
├── Dockerfile
└── docker-compose.yml           # optional, for local deploy
```

### Deployment (wb.alimov.top)

- Single Docker container behind reverse proxy (Caddy/nginx) with TLS
- Env: `PORT=8080`, `DB_PATH=/data/whiteboard.db`
- Persistent volume for `/data`
- WebSocket upgrade headers forwarded by proxy (`Upgrade`, `Connection`)

### Out of Scope (YAGNI)

- User accounts, permissions, room passwords
- Redis / Postgres / multi-instance
- Stroke compression, undo/redo, shapes, images
- Rate limiting (add later if abused)

---

## Implementation Tasks

### Task 1: Project Bootstrap

**Files:**
- Create: `go.mod`, `cmd/server/main.go`

**Step 1: Initialize module**

```bash
cd /Users/sasha/Desktop/git/public/whiteboard
go mod init github.com/alimov/whiteboard
```

**Step 2: Add dependencies**

```bash
go get github.com/gin-gonic/gin
go get github.com/gorilla/websocket
go get modernc.org/sqlite
```

**Step 3: Minimal main that starts Gin**

```go
// cmd/server/main.go
package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.String(200, "ok")
	})
	log.Fatal(r.Run(":" + port))
}
```

**Step 4: Verify**

```bash
go run ./cmd/server &
curl -s localhost:8080/health
# Expected: ok
kill %1
```

**Step 5: Commit**

```bash
git add go.mod go.sum cmd/server/main.go
git commit -m "chore: bootstrap gin server"
```

---

### Task 2: Session ID Generation

**Files:**
- Create: `internal/id/id.go`
- Create: `internal/id/id_test.go`

**Step 1: Write the failing test**

```go
// internal/id/id_test.go
package id

import (
	"regexp"
	"testing"
)

func TestGenerate(t *testing.T) {
	got, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 8 {
		t.Fatalf("want len 8, got %d: %q", len(got), got)
	}
	if !regexp.MustCompile(`^[A-Za-z0-9]{8}$`).MatchString(got) {
		t.Fatalf("invalid id: %q", got)
	}
}

func TestValid(t *testing.T) {
	if !Valid("a3Kf9xQp") {
		t.Fatal("expected valid")
	}
	if Valid("bad/id!") {
		t.Fatal("expected invalid")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/id/... -v
# Expected: FAIL — Generate not defined
```

**Step 3: Implement**

```go
// internal/id/id.go
package id

import (
	"crypto/rand"
	"math/big"
)

const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
const length = 8

var validRe = regexp.MustCompile(`^[A-Za-z0-9]{8}$`)

func Generate() (string, error) {
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		b[i] = alphabet[n.Int64()]
	}
	return string(b), nil
}

func Valid(s string) bool {
	return validRe.MatchString(s)
}
```

(Add `"regexp"` import.)

**Step 4: Run test to verify it passes**

```bash
go test ./internal/id/... -v
# Expected: PASS
```

**Step 5: Commit**

```bash
git add internal/id/
git commit -m "feat: add session id generation"
```

---

### Task 3: SQLite Storage Layer

**Files:**
- Create: `internal/storage/storage.go`
- Create: `internal/storage/sqlite.go`
- Create: `internal/storage/sqlite_test.go`

**Step 1: Write the failing test**

```go
// internal/storage/sqlite_test.go
package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteStorage(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	s, err := NewSQLite(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })

	if err := s.CreateSession(ctx, "a3Kf9xQp"); err != nil {
		t.Fatal(err)
	}
	exists, err := s.SessionExists(ctx, "a3Kf9xQp")
	if err != nil || !exists {
		t.Fatalf("session should exist: exists=%v err=%v", exists, err)
	}

	stroke := Stroke{Color: "#000", Width: 2, X0: 1, Y0: 2, X1: 3, Y1: 4}
	if err := s.AddStroke(ctx, "a3Kf9xQp", stroke); err != nil {
		t.Fatal(err)
	}
	strokes, err := s.ListStrokes(ctx, "a3Kf9xQp")
	if err != nil {
		t.Fatal(err)
	}
	if len(strokes) != 1 || strokes[0].X1 != 4 {
		t.Fatalf("unexpected strokes: %+v", strokes)
	}

	if err := s.ClearStrokes(ctx, "a3Kf9xQp"); err != nil {
		t.Fatal(err)
	}
	strokes, _ = s.ListStrokes(ctx, "a3Kf9xQp")
	if len(strokes) != 0 {
		t.Fatal("expected empty after clear")
	}
}
```

**Step 2: Run test to verify it fails**

```bash
go test ./internal/storage/... -v
# Expected: FAIL
```

**Step 3: Implement interface + SQLite**

```go
// internal/storage/storage.go
package storage

import "context"

type Stroke struct {
	Color string  `json:"color"`
	Width float64 `json:"width"`
	X0, Y0, X1, Y1 float64 `json:"x0","y0","x1","y1"`
}

type Storage interface {
	CreateSession(ctx context.Context, id string) error
	SessionExists(ctx context.Context, id string) (bool, error)
	AddStroke(ctx context.Context, sessionID string, s Stroke) error
	ListStrokes(ctx context.Context, sessionID string) ([]Stroke, error)
	ClearStrokes(ctx context.Context, sessionID string) error
	Close() error
}
```

Implement `NewSQLite`, migrations, and CRUD in `sqlite.go` using `database/sql` + `modernc.org/sqlite`. Serialize strokes as JSON in `payload` column.

**Step 4: Run test to verify it passes**

```bash
go test ./internal/storage/... -v
# Expected: PASS
```

**Step 5: Commit**

```bash
git add internal/storage/
git commit -m "feat: add sqlite session and stroke storage"
```

---

### Task 4: WebSocket Hub

**Files:**
- Create: `internal/hub/hub.go`
- Create: `internal/hub/hub_test.go`

**Step 1: Write the failing test**

Test that broadcasting to a room reaches connected clients (use httptest + gorilla/websocket or test the register/broadcast logic with mock conns).

Minimal test: register client, broadcast, verify message received on another client's channel.

**Step 2: Run test — expect FAIL**

**Step 3: Implement hub**

```go
// internal/hub/hub.go — core API
type Hub struct { /* rooms map[string]*Room */ }
func New() *Hub
func (h *Hub) Register(sessionID string, c *Client)
func (h *Hub) Unregister(sessionID string, c *Client)
func (h *Hub) Broadcast(sessionID string, msg []byte, except *Client)
```

Each `Client` holds `send chan []byte` and `conn *websocket.Conn`. One goroutine per client reads/writes.

**Step 4: Run test — expect PASS**

**Step 5: Commit**

```bash
git add internal/hub/
git commit -m "feat: add websocket room hub"
```

---

### Task 5: HTTP Handlers — Landing + Create/Join

**Files:**
- Create: `internal/handler/pages.go`
- Create: `internal/handler/session.go`
- Create: `web/templates/index.html`
- Create: `web/static/style.css`
- Modify: `cmd/server/main.go`

**Step 1: Write handler test**

```go
// internal/handler/session_test.go
func TestCreateSession(t *testing.T) {
	// setup gin test router with in-memory sqlite
	// POST /create → 302 redirect to /{8-char-id}
}
func TestJoinMissingSession(t *testing.T) {
	// POST /join with id=nonexist → 404
}
```

**Step 2: Run test — expect FAIL**

**Step 3: Implement**

- `index.html`: two forms — Create (POST `/create`), Join (POST `/join` with text input)
- `POST /create`: generate ID, `storage.CreateSession`, redirect 302 to `/{id}`
- `POST /join`: validate `id.Valid()`, check `SessionExists`, redirect or 404

Embed templates via `//go:embed web/templates/* web/static/*`.

**Step 4: Run test — expect PASS**

**Step 5: Manual check**

```bash
go run ./cmd/server
# open http://localhost:8080 — click Create, land on /xxxxxxxx
```

**Step 6: Commit**

```bash
git add internal/handler/ web/ cmd/server/main.go
git commit -m "feat: add landing page with create and join"
```

---

### Task 6: Whiteboard Page

**Files:**
- Create: `web/templates/board.html`
- Create: `web/static/board.js`
- Modify: `internal/handler/pages.go`

**Step 1: Implement GET `/{id}`**

- Reject reserved paths and invalid IDs → 404
- Check session exists → 404 if not
- Render `board.html` with session ID injected

**Step 2: board.html + board.js**

- Full-viewport `<canvas>`
- Toolbar: color picker, stroke width, Clear button
- Mouse/touch drawing → send stroke JSON over WebSocket
- Receive strokes/history → redraw

**Step 3: Manual check**

Open two browser tabs on same `/{id}`, draw in one, see it in the other.

**Step 4: Commit**

```bash
git add web/templates/board.html web/static/board.js internal/handler/pages.go
git commit -m "feat: add whiteboard canvas page"
```

---

### Task 7: WebSocket Handler

**Files:**
- Create: `internal/handler/websocket.go`
- Modify: `cmd/server/main.go`

**Step 1: Write integration test**

Connect WebSocket to `/ws/{id}`, receive `history`, send stroke, verify persistence via `storage.ListStrokes`.

**Step 2: Run test — expect FAIL**

**Step 3: Implement**

- Upgrade at `GET /ws/:id`
- Validate session exists
- Register with hub
- On connect: load strokes, send `{"type":"history","strokes":[...]}`
- On message: parse, persist (`AddStroke` / `ClearStrokes`), broadcast to room
- On disconnect: unregister

**Step 4: Run test — expect PASS**

**Step 5: Commit**

```bash
git add internal/handler/websocket.go cmd/server/main.go
git commit -m "feat: add websocket stroke sync"
```

---

### Task 8: Wire Everything in main.go

**Files:**
- Modify: `cmd/server/main.go`

**Step 1: Compose dependencies**

```go
ctx := context.Background()
store, _ := storage.NewSQLite(ctx, env("DB_PATH", "whiteboard.db"))
hub := hub.New()

r := gin.Default()
r.Static("/static", "web/static") // or embed
handler.RegisterRoutes(r, store, hub)
```

Route registration order: `/`, `/create`, `/join`, `/ws/:id`, then `/:id` last.

**Step 2: Smoke test**

```bash
go test ./... -v
go run ./cmd/server
# full flow: create → draw → restart server → rejoin → strokes still visible
```

**Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "chore: wire storage hub and handlers"
```

---

### Task 9: Docker + Deploy Notes

**Files:**
- Create: `Dockerfile`
- Create: `docker-compose.yml`
- Create: `README.md`

**Step 1: Multi-stage Dockerfile**

```dockerfile
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /whiteboard ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /whiteboard /whiteboard
ENV PORT=8080 DB_PATH=/data/whiteboard.db
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/whiteboard"]
```

**Step 2: docker-compose.yml with volume**

**Step 3: README with Caddy reverse proxy snippet for `wb.alimov.top`**

**Step 4: Commit**

```bash
git add Dockerfile docker-compose.yml README.md
git commit -m "docs: add docker and deployment instructions"
```

---

## Testing Checklist

- [ ] `go test ./...` passes
- [ ] Create session → unique 8-char URL
- [ ] Join nonexistent session → error shown
- [ ] Two tabs same room → strokes sync live
- [ ] Server restart → strokes reload from SQLite
- [ ] Clear button → all clients cleared + DB emptied
- [ ] Invalid/reserved paths → 404

## Future Enhancements (not in v1)

- Room expiry (TTL cleanup cron)
- Export board as PNG
- Simple rate limit on WebSocket messages
- HTTPS-only cookie for "recent rooms" list
