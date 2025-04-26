package repository

import (
	"database/sql"

	"GoSpeak/internal/model"

	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db *sqlx.DB
}

func (r *UserRepository) GetUser(id int64) (*model.User, error) {
	query := `
				SELECT user_id,
				       email,
				       full_name,
				       avatar_url,
				       is_online 
				FROM users 
				WHERE user_id=$1 ;
			`
	row := r.db.QueryRow(query, id)
	u := &model.User{}

	if err := row.Scan(&u.UserID, &u.Email, &u.FullName, &u.AvatarURL, &u.IsOnline); err != nil {
		return nil, err
	}
	return u, nil

}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

func (r *UserRepository) SignUp(u model.User) (*model.User, error) {
	query := `
			INSERT INTO users(email,password_hash, full_name) VALUES ($1,$2,$3) ;
			`
	_, err := r.db.Exec(query, u.Email, u.PasswordHash, u.FullName)
	if err != nil {
		return nil, err
	}
	return &u, nil

}

func (r *UserRepository) SignIn(email string, password string) (*model.User, error) {
	query := `
			SELECT user_id,email, full_name,avatar_url,is_online FROM users WHERE email = $1 AND password_hash = $2 ;
			`
	row := r.db.QueryRow(query, email, password)
	var u model.User
	if err := row.Scan(&u.UserID, &u.Email, &u.FullName, &u.AvatarURL, &u.IsOnline); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil

}

func (r *UserRepository) UpdateStatus(u *model.User) error {
	query := `
			UPDATE users SET is_online=$1 where user_id=$2
			`
	_, err := r.db.Exec(query, u.IsOnline, u.UserID)
	if err != nil {
		return err
	}
	return nil
}
