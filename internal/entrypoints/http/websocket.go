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
	for {
		var message *model.Message

		r.WebSocket.mu.Lock()
		r.WebSocket.clients[c] = true
		r.WebSocket.mu.Unlock()
		defer func() {
			r.WebSocket.mu.Lock()
			delete(r.WebSocket.clients, c)
			r.WebSocket.mu.Unlock()
			c.Close()
		}()
		_, msg, err := c.ReadMessage()
		if err != nil {
			slog.Error(err.Error())
			break
		}
		if err := json.Unmarshal(msg, &message); err != nil {
			slog.Error(err.Error())
			break
		}
		err = r.service.Message.Send(message)
		if err != nil {
			slog.Error(err.Error())
			continue
		}
		log.Printf("recv: %s", msg)
		r.WebSocket.broadcast <- msg

	}

}
func (r *Router) HandleWebSocketMessage() {
	for {
		msg := <-r.WebSocket.broadcast
		r.WebSocket.mu.Lock()
		for client := range r.WebSocket.clients {
			err := client.WriteMessage(websocket.TextMessage, msg)
			if err != nil {
				slog.Error(err.Error())
				client.Close()
				delete(r.WebSocket.clients, client)
			}
		}
	}
	r.WebSocket.mu.Unlock()

}

// ErrorResponse структура для ошибок
type ErrorResponse struct {
	Message string `json:"message"`
}

func isValidToken(token string) bool {
	// Реализуйте проверку токена
	return true
}
