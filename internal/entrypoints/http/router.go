package http

import (
	"sync"

	"GoSpeak/internal/config"
	"GoSpeak/internal/model"
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
	cfg     config.HostConfig
	*WebSocket
}
type WebSocket struct {
	mu        *sync.Mutex
	clients   map[*websocket.Conn]model.Participant
	broadcast chan Response
}
type Response struct {
	User_id      int64                   `json:"user_id"`
	Confrence_id string                  `json:"confrence_id"`
	Response     model.WebSocketResponse `json:"response"`
}

func NewRouter(service service.Service, cfg config.HostConfig) *Router {
	return &Router{service: service,
		cfg: cfg,
		WebSocket: &WebSocket{
			mu:        &sync.Mutex{},
			clients:   make(map[*websocket.Conn]model.Participant),
			broadcast: make(chan Response),
		},
	}
}

func (r *Router) Routes(app fiber.Router) {
	app.Static("assets", "web/assets")

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept, Content-Length, Accept-Language, Accept-Encoding, Connection, Access-Control-Allow-Origin",
		AllowMethods:     "GET, POST, HEAD, PUT, DELETE, PATCH, OPTIONS",
		AllowCredentials: false,
	}))
	app.Use(logger.New(logger.Config{
		Format: "[${ip}]:${port} ${status} - ${method} ${path} - ${ua}\\n\n",
	}))

	app.Get("/", r.IndexHandler)
	app.Get("/sign-in", r.RenderSignIn)
	app.Get("/sign-up", r.RenderSignUp)
	app.Get("/conference", r.RenderConference)
	auth := app.Group("/auth")
	user := app.Group("/user")
	conference := app.Group("/conference")
	auth.Post("/sign-up", r.PostSignUpHandler)
	auth.Post("/sign-in", r.PostSignInHandler)
	user.Get("/", r.GetUserHandler)
	conference.Use(r.JWTMiddleware)
	conference.Post("/create", r.CreateConferenceHandler)
	conference.Get("/join", r.JoinConferenceHandler)

	app.Use("/ws", r.WebsocketMiddleware)
	app.Get("/ws", websocket.New(r.HandleWebSocketConnection))

}

func (r *Router) NewPage() *Page {
	return &Page{Domain: r.cfg.Domain, Name: r.cfg.Name}
}
