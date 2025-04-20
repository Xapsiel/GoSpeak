package model

import (
	"encoding/json"
	"sync"

	"github.com/gofiber/websocket/v2"
)

type WebSocketMessage struct {
	Type         string          `json:"type"`
	ConferenceId string          `json:"conference_id"`
	SenderId     int64           `json:"sender_id"`
	TargetUserId int64           `json:"target_user_id"`
	Payload      json.RawMessage `json:"payload"`
}

type WebSocketResponse struct {
	Type         string          `json:"type"`
	Data         interface{}     `json:"data"`
	TargetUserId int64           `json:"target_user_id"`
	Payload      json.RawMessage `json:"payload"`
}

type Room struct {
	ID          string
	Participant map[*websocket.Conn]*Peer
	Mu          sync.RWMutex
}
