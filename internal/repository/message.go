package repository

import (
	"GoSpeak/internal/model"

	"github.com/jmoiron/sqlx"
)

type MessageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) *MessageRepository {

	return &MessageRepository{db: db}
}
func (r *MessageRepository) Send(m *model.Message) error {
	query := `
			INSERT INTO messages(conference_id, sender_id, content)
			VALUES ($1, $2, $3)	
		`
	_, err := r.db.Exec(query, m.ConferenceID, m.SenderID, m.Content)
	if err != nil {
		return err
	}
	return nil
}
