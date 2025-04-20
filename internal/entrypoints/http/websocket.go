package http

import (
	"encoding/json"
	"log/slog"

	"GoSpeak/internal/model"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
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
	r.WebSocket.clients[c] = &model.Participant{}
	r.WebSocket.peers[c] = &model.Peer{}
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
		var (
			currentPeer *model.Peer
		)
		response.TargetUserId = message.TargetUserId
		switch message.Type {
		case "join":

			response.Type = "user_joined" // Добавлено
			var m struct {
				UserId       int64  `json:"user_id"`
				ConferenceId string `json:"conference_id"`
				CreatorId    int64  `json:"creator_id"`
			}
			if err := json.Unmarshal(msg, &m); err != nil {
				slog.Error("Failed to parse join_conference message",
					"error", err,
					"raw", string(msg))
				break
			}
			slog.Info("Processing join_conference request",
				"participant_id", m.UserId,
				"conference_id", message.ConferenceId)
			r.mu.Lock()
			r.WebSocket.clients[c] = &model.Participant{
				ParticipantID: m.UserId,
				ConferenceID:  m.ConferenceId,
			}

			pc := r.service.WebRTC.CreatePeerConnection(r.rooms[m.ConferenceId], c)

			peer := &model.Peer{
				Conn: c,
				Pc:   pc,
			}
			r.WebSocket.peers[c] = peer
			r.WebSocket.rooms[m.ConferenceId].Participant[c] = peer
			r.mu.Unlock()

			slog.Debug("Adding participant to conference",
				"participant_id", m.UserId,
				"conference_id", m.ConferenceId)

			if err := r.service.Participant.AddToConference(m.UserId, &model.Conference{
				ConferenceID: m.ConferenceId,
				CreaterID:    m.CreatorId,
			}); err != nil {
				slog.Error("Failed to add participant to conference",
					"participant_id", m.UserId,
					"conference_id", m.ConferenceId,
					"error", err)
				break
			}
			currentPeer = peer

		//case "leave_conference":
		//	response.Type = "user_left"
		//	slog.Info("Processing leave_conference request",
		//		"participant_id", r.WebSocket.clients[c].ParticipantID,
		//		"conference_id", r.WebSocket.clients[c].ConferenceID)
		//
		//	participantID := r.WebSocket.clients[c].ParticipantID
		//
		//	if err := r.service.Participant.RemoveFromConference(participantID); err != nil {
		//		slog.Error("Failed to remove participant from conference",
		//			"participant_id", participantID,
		//			"error", err)
		//	}
		//
		//	response.Data = struct {
		//		User_id int64 `json:"user_id"`
		//	}{
		//		User_id: participantID,
		//	}

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

		case "offer":
			var offer webrtc.SessionDescription
			_ = json.Unmarshal(message.Payload, &offer)

			r.service.WebRTC.ReceiveOffer(currentPeer, offer)
		case "ice-candidate":
			var candidate webrtc.ICECandidateInit
			if err := json.Unmarshal(message.Payload, &candidate); err != nil {
				slog.Error("ICE candidate parsing failed", "error", err)
				continue
			}
			if err := r.service.WebRTC.ReceiveICECandidate(currentPeer, candidate); err != nil {
				slog.Error("ICE candidate processing failed", "error", err)
			}

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
			UserId:       r.WebSocket.clients[c].ParticipantID,
			ConferenceId: r.WebSocket.clients[c].ConferenceID,
			Response:     response,
		}
	}
}

func (r *Router) HandleWebSocketMessage() {
	slog.Info("Starting WebSocket message handler")
	defer slog.Info("Stopping WebSocket message handler")

	for msg := range r.WebSocket.broadcast {
		slog.Debug("Processing broadcast message",
			"type", msg.Response.Type,
			"conference_id", msg.ConferenceId,
			"sender_id", msg.UserId)

		r.WebSocket.mu.Lock()
		//clientsCount := len(r.WebSocket.clients)  // Не используется, можно удалить
		//slog.Debug("Active clients", "count", clientsCount) // Не используется, можно удалить

		room, ok := r.WebSocket.rooms[msg.ConferenceId]
		if !ok {
			slog.Warn("Conference not found for broadcast", "conference_id", msg.ConferenceId)
			r.WebSocket.mu.Unlock()
			continue // Перейти к следующему сообщению
		}

		for client, _ := range room.Participant {
			// Исключить отправителя из списка получателей (чтобы не отправлять сообщение обратно отправителю)
			if msg.UserId == r.WebSocket.clients[client].ParticipantID {
				slog.Debug("Skipping sender client", "client_id", msg.UserId)
				continue
			}

			// Отправить сообщение целевому пользователю или всем
			if msg.Response.TargetUserId == 0 || r.WebSocket.clients[client].ParticipantID == msg.Response.TargetUserId {
				slog.Debug("Sending message to client",
					"recipient", r.WebSocket.clients[client].ParticipantID,
					"message_type", msg.Response.Type)

				m, err := json.Marshal(msg)
				if err != nil {
					slog.Error("Failed to marshal message", "error", err)
					continue // Перейти к следующему клиенту
				}

				if err := client.WriteMessage(websocket.TextMessage, m); err != nil {
					slog.Error("Failed to send message",
						"recipient", r.WebSocket.clients[client].ParticipantID,
						"error", err)
					client.Close()
					delete(r.WebSocket.clients, client)

					// Удалить участника из комнаты и списка клиентов, если отправка сообщения не удалась
					delete(room.Participant, client)
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
