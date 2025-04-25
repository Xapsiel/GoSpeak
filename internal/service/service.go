package service

import (
	"GoSpeak/internal/model"
	"GoSpeak/internal/repository"
)

const (
	ErrNotFound = "not found"
)

type Service struct {
	User
	Conference
	Participant
	Message
}

func NewService(repo repository.Repository) *Service {
	return &Service{
		User:        NewUserService(repo),
		Conference:  NewConferenceService(repo),
		Participant: NewParticipantService(repo),
		Message:     NewMessageService(repo),
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
	AddToConference(u int64, conf string) error
	RemoveFromConference(id int64) error
	GetParticipantsByConferenceID(id string) ([]int64, error)
}

type Message interface {
	Send(m *model.Message) error
}
