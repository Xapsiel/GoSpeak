package http

import (
	"encoding/json"
	"log"
	"log/slog"

	"GoSpeak/internal/model"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

func (r *Router) WebsocketMiddleware(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}
func (r *Router) HandleWebSocketConnection(c *websocket.Conn) {
	r.WebSocket.mu.Lock()
	r.WebSocket.clients[c] = (model.Participant{})
	r.WebSocket.mu.Unlock()
	for {
		var message *model.WebSocketMessage = new(model.WebSocketMessage)
		var response *model.WebSocketResponse = &model.WebSocketResponse{}

		defer func() {
			r.WebSocket.mu.Lock()
			if _, ok := r.WebSocket.clients[c]; ok {
				delete(r.WebSocket.clients, c)
				c.Close()

			}
			r.WebSocket.mu.Unlock()
		}()
		_, msg, err := c.ReadMessage()
		if err != nil {
			err = r.service.Participant.RemoveFromConference(r.clients[c].ParticipantID)
			if err != nil {
				slog.Error(err.Error())
			}
			break
		}
		if err := json.Unmarshal(msg, &message); err != nil {
			slog.Error(err.Error())
			break
		}
		switch message.Type {
		case "join_conference":
			response.Type = "user_joined"
			var m struct {
				User_id       int64  `json:"user_id"`
				Conference_id string `json:"conference_id"`
				Creater_id    int64  `json:"creater_id"`
			}
			if err := json.Unmarshal(msg, &m); err != nil {
				slog.Error(err.Error())
				break
			}
			r.clients[c] = model.Participant{ParticipantID: m.User_id, ConferenceID: m.Conference_id}

			err := r.service.Participant.AddToConference(m.User_id, &model.Conference{ConferenceID: m.Conference_id, CreaterID: m.Creater_id})
			if err != nil {
				slog.Error(err.Error())
				break
			}
			response.Data = struct {
				User_id int64 `json:"user_id"`
			}{
				User_id: m.User_id,
			}

		case "leave_conference":
			response.Type = "user_left"
		case "chat_message":
			response.Type = "new_message"
			var m *model.Message
			if err := json.Unmarshal(msg, &m); err != nil {
				slog.Error(err.Error())
				break
			}
			err = r.service.Message.Send(m)
			if err != nil {
				slog.Error(err.Error())
				continue
			}
			response.Data = *m

			log.Printf("recv: %s", msg)
		case "send_offer":
			response.Type = "receive_offer"
			var offer struct {
				Offer interface{} `json:"offer"`
			}
			if err := json.Unmarshal(msg, &offer); err != nil {
				slog.Error(err.Error())
				break
			}
			response.Data = offer.Offer

		case "send_answer":
			response.Type = "receive_answer"
			var answer struct {
				Answer interface{} `json:"answer"`
			}
			if err := json.Unmarshal(msg, &answer); err != nil {
				slog.Error(err.Error())
				break
			}
			response.Data = answer

		case "send_ice_candidate":
			response.Type = "receive_ice_candidate"
			var ice struct {
				Candidate interface{} `json:"candidate"`
			}
			if err := json.Unmarshal(msg, &ice); err != nil {
				slog.Error(err.Error())
				break
			}
			response.Data = ice.Candidate

		}
		r.WebSocket.broadcast <- Response{
			User_id:      r.clients[c].ParticipantID,
			Confrence_id: r.clients[c].ConferenceID,
			Response:     *response,
		}

	}

}
func (r *Router) HandleWebSocketMessage() {
	for {
		msg := <-r.WebSocket.broadcast
		r.WebSocket.mu.Lock()
		for client := range r.WebSocket.clients {
			if r.WebSocket.clients[client].ParticipantID != msg.User_id && r.WebSocket.clients[client].ConferenceID == msg.Confrence_id {
				m, _ := json.Marshal(msg)
				err := client.WriteMessage(websocket.TextMessage, m)
				if err != nil {
					slog.Error(err.Error())
					client.Close()
					delete(r.WebSocket.clients, client)
				}
			}

		}
		r.WebSocket.mu.Unlock()

	}

}

type ErrorResponse struct {
	Message string `json:"message"`
}
