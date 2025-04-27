package repository

import (
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

var UserInAnotherErr = errors.New("User in another conference")

type ParticipantRepository struct {
	db *sqlx.DB
}

func NewParticipantRepository(db *sqlx.DB) *ParticipantRepository {
	return &ParticipantRepository{
		db: db,
	}
}

func (r *ParticipantRepository) IsUserInConf(u int64) ([]string, error) {
	query := `
				SELECT c.conference_id FROM participants
				LEFT JOIN conferences as c
				         ON c.conference_id=participants.conference_id
				         WHERE user_id=$1
				`
	ids := make([]string, 0)
	var id string
	rows, err := r.db.Query(query, u)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}
func (r *ParticipantRepository) AddToConference(u int64, conf string) error {
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
	_, err := r.db.Exec(query, conf, u, role, time.Now())
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
func (r *ParticipantRepository) GetParticipantsByConferenceID(id string) ([]int64, error) {
	query := `

		SELECT user_id FROM participants WHERE conference_id = $1;
	`
	rows, err := r.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	participants := make([]int64, 0)
	for rows.Next() {
		if rows.Err() != nil {
			return nil, err
		}
		var id int64 = 0
		rows.Scan(&id)
		participants = append(participants, id)

	}
	return participants, nil
}
