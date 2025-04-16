package model

type WebSocketMessage struct {
	Type         string `json:"type"`
	ConferenceId string `json:"conference_id"`
	SenderId     int64  `json:"sender_id"`
	TargetUserId int64  `json:"target_user_id"`
}

type WebSocketResponse struct {
	Type         string      `json:"type"`
	Data         interface{} `json:"data"`
	TargetUserId int64       `json:"target_user_id"`
}
