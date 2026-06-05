package handler

import (
	"github.com/gorilla/websocket"
)

func websocketDial(url string) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	return conn, err
}
