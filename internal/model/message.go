package model

import "time"

type Message struct {
	MessageID    int64     `json:"message_id"`
	ConferenceID string    `json:"conference_id"`
	SenderID     int64     `json:"sender_id"`
	Content      string    `json:"content"`
	SentAt       time.Time `json:"sent_at"`
	ContentType  string    `json:"content_type"`
	FileURL      string    `json:"file_url, omitempty"`
}
