package repository

import (
	"time"

	"GoSpeak/internal/model"

	"github.com/jmoiron/sqlx"
)

type ParticipantRepository struct {
	db *sqlx.DB
}

func NewParticipantRepository(db *sqlx.DB) *ParticipantRepository {
	return &ParticipantRepository{
		db: db,
	}
}

func (r *ParticipantRepository) AddToConference(u int64, conf *model.Conference) error {
	query := `

		INSERT INTO 
		    participants(conference_id, user_id, role, joined_at)
		VALUES
		    ($1, $2, $3, $4)
		ON CONFLICT
		    (user_id) 
		DO UPDATE
		    SET role = EXCLUDED.role, joined_at = EXCLUDED.joined_at, 
		        conference_id = EXCLUDED.conference_id, user_id = EXCLUDED.user_id;
	
			`
	role := "participant"
	if (u) == conf.CreaterID {
		role = "host"
	}
	_, err := r.db.Exec(query, conf.ConferenceID, u, role, time.Now())
	if err != nil {
		return err
	}
	return nil

}
func (r *ParticipantRepository) RemoveFromConference(id int64) error {
	query :=
		`DELETE FROM participants WHERE user_id = $1;`
	_, err := r.db.Exec(query, id)
	return err

}
