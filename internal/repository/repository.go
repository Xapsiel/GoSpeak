package repository

import (
	"GoSpeak/internal/model"

	"github.com/jmoiron/sqlx"
)

type Repository struct {
	User
	Conference
	Participant
	Message
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		User:        NewUserRepository(db),
		Conference:  NewConferenceRepository(db),
		Message:     NewMessageRepository(db),
		Participant: NewParticipantRepository(db),
	}
}

type User interface {
	SignUp(u model.User) (*model.User, error)
	SignIn(email string, password string) (*model.User, error)
	GetUser(id int64) (*model.User, error)
	UpdateStatus(u *model.User) error
}
type Conference interface {
	CreateConference(c *model.Conference) error
	GetConference(join_url string) (*model.Conference, error)
}
type Participant interface {
	AddToConference(u int64, conf string) error
	RemoveFromConference(id int64) error
	GetParticipantsByConferenceID(id string) ([]int64, error)
	IsUserInConf(u int64) ([]string, error)
}

type Message interface {
	Send(m *model.Message) error
}
