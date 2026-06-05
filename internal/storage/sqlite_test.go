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
	if len(strokes) != 1 || strokes[0].X1 != 3 {
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
