package model

import "time"

type Participant struct {
	ParticipantID int64     `json:"participant_id"`
	ConferenceID  string    `json:"conference_id"`
	UserID        int64     `json:"user_id"`
	Role          string    `json:"role"`
	JoinedAt      time.Time `json:"joined_at"`
	LeftAt        time.Time `json:"left_at"`
}
