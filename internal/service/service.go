package service

import (
	"GoSpeak/internal/model"
	"GoSpeak/internal/repository"

	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
)

const (
	ErrNotFound = "not found"
)

type Service struct {
	User
	Conference
	Participant
	Message
	WebRTC
}

func NewService(repo repository.Repository) *Service {
	return &Service{
		User:        NewUserService(repo),
		Conference:  NewConferenceService(repo),
		Participant: NewParticipantService(repo),
		Message:     NewMessageService(repo),
		WebRTC:      NewWebRTCService(repo),
	}
}

type User interface {
	SignUp(u *model.SignUpUser) (*model.User, error)
	SignIn(u *model.SignUpUser) (*model.User, string, error)
	ParseJWT(tokenstring string) (int64, error)
	GetUser(id int64) (*model.User, error)
	UpdateStatus(u *model.User) error
}
type Conference interface {
	CreateConference(c *model.Conference) (*model.Conference, error)
	GetConference(joinUrl string) (*model.Conference, error)
}

type Participant interface {
	AddToConference(u int64, conf *model.Conference) error
	RemoveFromConference(id int64) error
	GetParticipantsByConferenceID(id string) ([]int64, error)
}

type Message interface {
	Send(m *model.Message) error
}

type WebRTC interface {
	CreatePeerConnection(room *model.Room, conn *websocket.Conn) *webrtc.PeerConnection
	RelayTrack(remote *webrtc.TrackRemote, local *webrtc.TrackLocalStaticRTP)
	ReceiveOffer(peer *model.Peer, offer webrtc.SessionDescription) error
	ReceiveICECandidate(peer *model.Peer, candidate webrtc.ICECandidateInit) error
}
