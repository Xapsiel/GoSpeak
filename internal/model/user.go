package model

type User struct {
	UserID       int64  `json:"user_id"`
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
	FullName     string `json:"full_name"`
	AvatarURL    string `json:"avatar_url"`
	CreatedAt    string `json:"-"`
	LastLogin    string `json:"-"`
	IsOnline     bool   `json:"is_online"`
}

type SignUpUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
