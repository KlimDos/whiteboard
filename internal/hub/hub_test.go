package hub

import (
	"testing"
	"time"
)

func TestBroadcast(t *testing.T) {
	h := New()
	c1 := &Client{Send: make(chan []byte, 8)}
	c2 := &Client{Send: make(chan []byte, 8)}

	h.Register("room1", c1)
	h.Register("room1", c2)

	msg := []byte(`{"type":"stroke"}`)
	h.Broadcast("room1", msg, c1)

	select {
	case got := <-c2.Send:
		if string(got) != string(msg) {
			t.Fatalf("unexpected message: %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}

	select {
	case <-c1.Send:
		t.Fatal("sender should be excluded")
	default:
	}

	h.Unregister("room1", c1)
	h.Unregister("room1", c2)
}
