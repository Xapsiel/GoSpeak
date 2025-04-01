package model

type WebSocketMessage struct {
	Type         string `json:"type"`
	ConferenceId string `json:"conference_id"`
}

type WebSocketResponse struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}
