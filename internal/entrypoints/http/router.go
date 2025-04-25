package http

import (
	"sync"
	"time"

	"GoSpeak/internal/config"
	"GoSpeak/internal/model"
	"GoSpeak/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
)

const (
	ErrNotFound = "not found"
)

type Router struct {
	service    service.Service
	cfg        config.HostConfig
	roomslock  sync.Mutex
	clientlock sync.Mutex
	*WebSocket
	Conference map[string]*model.Conference
}
type WebSocket struct {
	rooms   map[string]*Room
	clients map[string]*ChatRoom
}
type websocketStreamerMessage struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}
type websocketChatMessage struct {
	Event        string `json:"event"`
	Data         string `json:"data"`
	From         int64  `json:"from"`
	ConferenceID string `json:"conference_id"`
}
type peerConnectionState struct {
	peerConnection *webrtc.PeerConnection
	websocket      *threadSafeWriter
}
type Room struct {
	listLock         sync.RWMutex
	signalDebounceMU sync.Mutex
	pendingSignal    bool
	peerConnections  []peerConnectionState
	trackLocals      map[string]*webrtc.TrackLocalStaticRTP
	lastPLI          time.Time
}
type ChatRoom struct {
	listlock sync.RWMutex
	conn     map[*websocket.Conn]*threadSafeWriter
}
type threadSafeWriter struct {
	*websocket.Conn
	sync.Mutex
	UserID int64
}
type RoomStats struct {
	Participants int
	Bitrate      uint64
	CPUUsage     float64
	TotalBitrate float64
}

func (t *threadSafeWriter) WriteJSON(v interface{}) error {
	t.Lock()
	defer t.Unlock()

	return t.Conn.WriteJSON(v)
}

func NewRouter(service service.Service, cfg config.HostConfig) *Router {
	return &Router{service: service,
		cfg: cfg,
		WebSocket: &WebSocket{
			rooms:   make(map[string]*Room),
			clients: make(map[string]*ChatRoom),
		},
	}
}

func (r *Router) Routes(app fiber.Router) {
	app.Static("assets", "web/assets")

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://192.168.0.100:3000, http://127.0.0.1:3000",
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		AllowCredentials: true,
		ExposeHeaders:    "Content-Length",
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

	app.Get("/ws/stream", websocket.New(r.WebSocketStreamerHandler))
	app.Get("/ws/chat", websocket.New(r.WebSocketChatHandler))
	go r.cleanupRooms()
}

func (r *Router) NewPage() *Page {
	return &Page{Domain: r.cfg.Domain, Name: r.cfg.Name}
}

func (r *Router) HandleWebSocketConnection(conn *websocket.Conn) {

}
