package http

import (
	"sync"

	"GoSpeak/internal/config"
	"GoSpeak/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

const (
	ErrNotFound = "not found"
)

type Router struct {
	service service.Service
	Domain  string
	*WebSocket
}
type WebSocket struct {
	mu        *sync.Mutex
	clients   map[*websocket.Conn]bool
	broadcast chan []byte
}

func NewRouter(service service.Service, cfg config.HostConfig) *Router {
	return &Router{service: service,
		Domain: cfg.Domain,
		WebSocket: &WebSocket{
			mu:        &sync.Mutex{},
			clients:   make(map[*websocket.Conn]bool),
			broadcast: make(chan []byte),
		},
	}
}

func (r *Router) Routes(app fiber.Router) {
	app.Static("assets", "web/assets")
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowCredentials: false,
	}))
	app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path} - ${ua}\\n\n",
	}))

	// Роуты аутентификации
	auth := app.Group("/auth")
	user := app.Group("/user")
	conference := app.Group("/conference")
	auth.Post("/sign-up", r.PostSignUpHandler)
	auth.Post("/sign-in", r.PostSignInHandler)
	user.Get("/", r.GetUserHandler)
	app.Get("/", r.IndexHandler)
	conference.Use(r.JWTMiddleware)
	conference.Post("/create", r.CreateConferenceHandler)
	conference.Get("/join", r.JoinConferenceHandler)

	app.Use("/ws", r.WebsocketMiddleware)
	app.Get("/ws", websocket.New(r.HandleWebSocketConnection))

}

func (r *Router) NewPage() *Page {
	return &Page{Domain: r.Domain}
}
