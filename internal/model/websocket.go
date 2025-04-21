package model

import (
	"sync"

	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
)

type WebsocketMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

type PeerConnectionState struct {
	peerConnection *webrtc.PeerConnection
	websocket      *ThreadSafeWriter
}
type ThreadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
}

func (t *ThreadSafeWriter) WriteJSON(v interface{}) error {
	t.Lock()
	defer t.Unlock()

	return t.Conn.WriteJSON(v)
}
