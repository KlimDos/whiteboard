package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS sessions (
    id         TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS strokes (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    payload    TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_strokes_session ON strokes(session_id, id);
`

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLite(ctx context.Context, path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, err
	}
	return &SQLiteStorage{db: db}, nil
}

func (s *SQLiteStorage) CreateSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sessions (id) VALUES (?)`, id)
	return err
}

func (s *SQLiteStorage) SessionExists(ctx context.Context, id string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM sessions WHERE id = ?`, id).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *SQLiteStorage) AddStroke(ctx context.Context, sessionID string, stroke Stroke) error {
	payload, err := json.Marshal(stroke)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO strokes (session_id, payload) VALUES (?, ?)`, sessionID, string(payload))
	return err
}

func (s *SQLiteStorage) ListStrokes(ctx context.Context, sessionID string) ([]Stroke, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT payload FROM strokes WHERE session_id = ? ORDER BY id`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var strokes []Stroke
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var stroke Stroke
		if err := json.Unmarshal([]byte(payload), &stroke); err != nil {
			return nil, fmt.Errorf("decode stroke: %w", err)
		}
		strokes = append(strokes, stroke)
	}
	return strokes, rows.Err()
}

func (s *SQLiteStorage) ClearStrokes(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM strokes WHERE session_id = ?`, sessionID)
	return err
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}
