package storage

import "context"

type Stroke struct {
	Color          string  `json:"color"`
	Width          float64 `json:"width"`
	X0             float64 `json:"x0"`
	Y0             float64 `json:"y0"`
	X1             float64 `json:"x1"`
	Y1             float64 `json:"y1"`
}

type Storage interface {
	CreateSession(ctx context.Context, id string) error
	SessionExists(ctx context.Context, id string) (bool, error)
	AddStroke(ctx context.Context, sessionID string, s Stroke) error
	ListStrokes(ctx context.Context, sessionID string) ([]Stroke, error)
	ClearStrokes(ctx context.Context, sessionID string) error
	Close() error
}
