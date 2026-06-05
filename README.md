# Whiteboard

Minimal real-time whiteboard service. Create or join a session at `/`, draw on a shared canvas synced over WebSocket. Strokes persist in SQLite.

## Run locally

```bash
go run ./cmd/server
```

Open http://localhost:8080

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `DB_PATH` | `whiteboard.db` | SQLite database path |

## Docker

```bash
docker compose up --build
```

## Deploy (wb.alimov.top)

Run the container with a persistent volume at `/data`. Put Caddy or nginx in front with TLS and WebSocket upgrade support.

Example Caddyfile:

```
wb.alimov.top {
    reverse_proxy localhost:8080
}
```

Caddy handles WebSocket upgrades automatically.

## API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Landing page |
| POST | `/create` | Create session, redirect to `/{id}` |
| POST | `/join` | Join by 8-char session ID |
| GET | `/{id}` | Whiteboard page |
| GET | `/ws/{id}` | WebSocket stroke sync |

## Tests

```bash
go test ./...
```
