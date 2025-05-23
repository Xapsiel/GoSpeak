package repository

import (
	"fmt"

	"GoSpeak/internal/model"

	"github.com/jmoiron/sqlx"
)

type ConferenceRepository struct {
	db *sqlx.DB
}

func (r *ConferenceRepository) DeleteConference(url string) error {
	query := `
			DELETE  FROM conferences
			WHERE join_url=$1;
			`
	_, err := r.db.Exec(query, url)
	if err != nil {
		return err
	}
	return nil
}

func NewConferenceRepository(db *sqlx.DB) *ConferenceRepository {
	return &ConferenceRepository{db: db}

}

func (r *ConferenceRepository) CreateConference(c *model.Conference) error {
	query := `
			INSERT INTO 
			    conferences(title, description, creator_id, status, join_url, password)
			VALUES ($1, $2, $3, $4,$5,$6)
			ON CONFLICT (title, description,creator_id) DO NOTHING 
			`
	_, err := r.db.Exec(query, c.Title, c.Description, c.CreatorID, "live", c.JoinURL, c.Password)
	if err != nil {
		return fmt.Errorf("Error with creating conference: %v", err)
	}
	return nil
}

func (r *ConferenceRepository) GetConference(join_url string) (*model.Conference, error) {
	query := `
			SELECT 
			    conference_id,
			    title,
			    description,
			    creator_id,
			    status,
			    join_url
			from conferences
			WHERE 
			    join_url = $1 and status='live';
			`
	res := r.db.QueryRow(query, join_url)
	var conf model.Conference
	err := res.Scan(&conf.ConferenceID, &conf.Title, &conf.Description, &conf.CreatorID, &conf.Status, &join_url)
	if err != nil {
		return nil, fmt.Errorf("Error getting conference: %v", err)
	}
	conf.JoinURL = join_url
	return &conf, nil

}
