package model

import "time"

type Conference struct {
	ConferenceID    string    `json:"conference_id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	CreaterID       int64     `json:"creater_id"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	Status          string    `json:"status"`
	JoinURL         string    `json:"join_url"`
	Password        string    `json:"password"`
	MaxParticipants int64     `json:"max_participants"`
}

type CreateConference struct {
	Title    string `json:"title"`
	Password string `json:"password"`
}
