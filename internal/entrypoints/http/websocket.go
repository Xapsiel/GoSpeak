package http

import (
	"encoding/json"
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
	// Логирование нового подключения
	slog.Info("New WebSocket connection",
		"remote_addr", c.RemoteAddr().String(),
		"local_addr", c.LocalAddr().String())

	r.WebSocket.mu.Lock()
	r.WebSocket.clients[c] = model.Participant{}
	r.WebSocket.mu.Unlock()

	defer func() {
		r.WebSocket.mu.Lock()
		defer r.WebSocket.mu.Unlock()
		if participant, ok := r.WebSocket.clients[c]; ok {
			slog.Info("Closing connection",
				"participant_id", participant.ParticipantID,
				"conference_id", participant.ConferenceID)
			delete(r.WebSocket.clients, c)
			c.Close()
		}
	}()

	for {
		var message *model.WebSocketMessage = new(model.WebSocketMessage)
		var response model.WebSocketResponse = model.WebSocketResponse{}

		_, msg, err := c.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err) {
				slog.Warn("Unexpected close error",
					"error", err,
					"participant_id", r.WebSocket.clients[c].ParticipantID)
			}
			participantID := r.WebSocket.clients[c].ParticipantID
			if err := r.service.Participant.RemoveFromConference(participantID); err != nil {
				slog.Error("Failed to remove participant from conference",
					"participant_id", participantID,
					"error", err)
			}
			break
		}

		slog.Debug("Received message",
			"raw", string(msg),
			"participant_id", r.WebSocket.clients[c].ParticipantID)

		if err := json.Unmarshal(msg, &message); err != nil {
			slog.Error("Failed to unmarshal message",
				"error", err,
				"raw", string(msg))
			break
		}

		response.TargetUserId = message.TargetUserId

		switch message.Type {
		case "join_conference":
			response.Type = "user_joined" // Добавлено
			var m struct {
				User_id       int64  `json:"user_id"`
				Conference_id string `json:"conference_id"`
				Creater_id    int64  `json:"creater_id"`
			}
			if err := json.Unmarshal(msg, &m); err != nil {
				slog.Error("Failed to parse join_conference message",
					"error", err,
					"raw", string(msg))
				break
			}
			slog.Info("Processing join_conference request",
				"participant_id", m.User_id,
				"conference_id", message.ConferenceId)

			r.WebSocket.clients[c] = model.Participant{
				ParticipantID: m.User_id,
				ConferenceID:  m.Conference_id,
			}

			slog.Debug("Adding participant to conference",
				"participant_id", m.User_id,
				"conference_id", m.Conference_id)

			if err := r.service.Participant.AddToConference(m.User_id, &model.Conference{
				ConferenceID: m.Conference_id,
				CreaterID:    m.Creater_id,
			}); err != nil {
				slog.Error("Failed to add participant to conference",
					"participant_id", m.User_id,
					"conference_id", m.Conference_id,
					"error", err)
				break
			}

			response.Data = struct {
				User_id int64 `json:"user_id"`
			}{
				User_id: m.User_id,
			}

		case "leave_conference":
			response.Type = "user_left"
			slog.Info("Processing leave_conference request",
				"participant_id", r.WebSocket.clients[c].ParticipantID,
				"conference_id", r.WebSocket.clients[c].ConferenceID)

			participantID := r.WebSocket.clients[c].ParticipantID

			if err := r.service.Participant.RemoveFromConference(participantID); err != nil {
				slog.Error("Failed to remove participant from conference",
					"participant_id", participantID,
					"error", err)
			}

			response.Data = struct {
				User_id int64 `json:"user_id"`
			}{
				User_id: participantID,
			}

		case "chat_message":
			response.Type = "new_message"
			var m model.Message
			if err := json.Unmarshal(msg, &m); err != nil {
				slog.Error("Failed to parse chat message",
					"error", err,
					"raw", string(msg))
				break
			}

			slog.Info("Processing chat message",
				"sender_id", m.SenderID,
				"conference_id", m.ConferenceID)

			if err := r.service.Message.Send(&m); err != nil {
				slog.Error("Failed to save message",
					"error", err,
					"message_id", m.MessageID)
				break
			}

			response.Data = m

		case "send_offer":
			response.Type = "receive_offer"
			var offer struct {
				TargetUserId int64       `json:"target_user_id"`
				Offer        interface{} `json:"offer"`
			}
			if err := json.Unmarshal(msg, &offer); err != nil {
				slog.Error("Failed to parse WebRTC offer",
					"error", err,
					"raw", string(msg))
				break
			}

			slog.Debug("Forwarding WebRTC offer",
				"from", r.WebSocket.clients[c].ParticipantID,
				"to", offer.TargetUserId)

			response.Data = struct {
				Sender_id int64       `json:"sender_id"`
				Offer     interface{} `json:"offer"`
			}{
				Sender_id: r.WebSocket.clients[c].ParticipantID,
				Offer:     offer.Offer,
			}
			response.TargetUserId = offer.TargetUserId

		case "send_answer":
			response.Type = "receive_answer"
			var answer struct {
				TargetUserId int64       `json:"target_user_id"`
				Answer       interface{} `json:"answer"`
			}
			if err := json.Unmarshal(msg, &answer); err != nil {
				slog.Error("Failed to parse WebRTC answer",
					"error", err,
					"raw", string(msg))
				break
			}

			slog.Debug("Forwarding WebRTC answer",
				"from", r.WebSocket.clients[c].ParticipantID,
				"to", answer.TargetUserId)

			response.Data = struct {
				Sender_id int64       `json:"sender_id"`
				Answer    interface{} `json:"answer"`
			}{
				Sender_id: r.WebSocket.clients[c].ParticipantID,
				Answer:    answer.Answer,
			}
			response.TargetUserId = answer.TargetUserId

		case "send_ice_candidate":
			response.Type = "receive_ice_candidate"
			var ice struct {
				TargetUserId int64       `json:"target_user_id"`
				Candidate    interface{} `json:"candidate"`
			}
			if err := json.Unmarshal(msg, &ice); err != nil {
				slog.Error("Failed to parse ICE candidate",
					"error", err,
					"raw", string(msg))
				break
			}

			slog.Debug("Forwarding ICE candidate",
				"from", r.WebSocket.clients[c].ParticipantID,
				"to", ice.TargetUserId)

			response.Data = struct {
				Sender_id int64       `json:"sender_id"`
				Candidate interface{} `json:"candidate"`
			}{
				Sender_id: r.WebSocket.clients[c].ParticipantID,
				Candidate: ice.Candidate,
			}
			response.TargetUserId = ice.TargetUserId

		case "request_participants":
			response.Type = "participants_list" // Исправлено с request_participants
			var req struct {
				ConferenceId string `json:"conference_id"`
			}
			if err := json.Unmarshal(msg, &req); err != nil {
				slog.Error("Failed to parse participants request",
					"error", err,
					"raw", string(msg))
				break
			}

			slog.Info("Processing participants request",
				"conference_id", req.ConferenceId,
				"requested_by", r.WebSocket.clients[c].ParticipantID)

			participants, err := r.service.Participant.GetParticipantsByConferenceID(req.ConferenceId)
			if err != nil {
				slog.Error("Failed to get participants",
					"conference_id", req.ConferenceId,
					"error", err)
				break
			}

			slog.Debug("Retrieved participants",
				"count", len(participants),
				"conference_id", req.ConferenceId)

			response.Data = participants
			response.TargetUserId = r.WebSocket.clients[c].ParticipantID

		default:
			slog.Warn("Unknown message type received",
				"type", message.Type,
				"participant_id", r.WebSocket.clients[c].ParticipantID)
			continue
		}
		slog.Debug("Sending response",
			"type", response.Type,
			"target_user", response.TargetUserId,
			"sender", r.WebSocket.clients[c].ParticipantID)

		r.WebSocket.broadcast <- Response{
			User_id:       r.WebSocket.clients[c].ParticipantID,
			Conference_id: r.WebSocket.clients[c].ConferenceID,
			Response:      response,
		}
	}
}

func (r *Router) HandleWebSocketMessage() {
	slog.Info("Starting WebSocket message handler")
	defer slog.Info("Stopping WebSocket message handler")

	for msg := range r.WebSocket.broadcast {
		slog.Debug("Processing broadcast message",
			"type", msg.Response.Type,
			"conference_id", msg.Conference_id,
			"sender_id", msg.User_id)

		r.WebSocket.mu.Lock()
		clientsCount := len(r.WebSocket.clients)
		slog.Debug("Active clients",
			"count", clientsCount)

		for client := range r.WebSocket.clients {
			if msg.Response.TargetUserId == 0 || r.WebSocket.clients[client].ParticipantID == msg.Response.TargetUserId {

				slog.Debug("Sending message to client",
					"recipient", r.WebSocket.clients[client].ParticipantID,
					"message_type", msg.Response.Type)

				m, _ := json.Marshal(msg)
				if err := client.WriteMessage(websocket.TextMessage, m); err != nil {
					slog.Error("Failed to send message",
						"recipient", r.WebSocket.clients[client].ParticipantID,
						"error", err)
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
