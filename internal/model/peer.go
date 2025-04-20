package model

import (
	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
)

type Peer struct {
	Conn        *websocket.Conn
	Pc          *webrtc.PeerConnection
	IsPublisher bool
}
