package service

import (
	"fmt"
	"net/mail"
	"time"

	"GoSpeak/internal/model"
	"GoSpeak/internal/repository"
	"GoSpeak/internal/utils"

	"github.com/dgrijalva/jwt-go"
)

type UserService struct {
	repo repository.User
}

const (
	signingKey = ("afgkasogdnasgvuio2r1jioqwdf89zsfiolkasf")
)

type tokenClaims struct {
	jwt.StandardClaims
	UserId int64  `json:"user_id"`
	Login  string `json:"login"`
}

func NewUserService(repo repository.User) *UserService {
	return &UserService{
		repo: repo,
	}
}

func (s *UserService) SignUp(u *model.SignUpUser) (*model.User, error) {
	_, err := mail.ParseAddress(u.Email)
	if err != nil {
		return nil, fmt.Errorf("Invalid mail address: %s", u.Email)
	}
	err = utils.ValidateLogin(u.Name)
	if err != nil {
		return nil, err
	}
	if err = utils.ValidatePassword(u.Password); err != nil {
		return nil, fmt.Errorf("Invalid password format: %s", u.Password)
	}
	var user model.User
	user.PasswordHash = utils.GeneratePasswordHash(u.Password)
	user.Email = u.Email
	user.FullName = u.Name
	return s.repo.SignUp(user)
}

func (s *UserService) SignIn(u *model.SignUpUser) (*model.User, string, error) {
	user, err := s.repo.SignIn(u.Email, utils.GeneratePasswordHash(u.Password))
	if err != nil {
		return nil, "", fmt.Errorf("SignIn Error: %v", err)
	}
	if user == nil {
		return user, "", nil
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &tokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Hour * 24 * 10).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
		UserId: user.UserID,
		Login:  user.FullName,
	})
	if user.AvatarURL == "" {
		user.AvatarURL = "/assets/static/images/avatar.png"
	}
	tokenString, err := token.SignedString([]byte(signingKey))
	if err != nil {
		return nil, "", fmt.Errorf("error creating the jwt token")
	}
	return user, tokenString, nil
}

func (s *UserService) ParseJWT(tokenstring string) (int64, error) {
	token, err := jwt.ParseWithClaims(tokenstring, &tokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(signingKey), nil
	})
	if err != nil {
		return 0, err
	}
	if claims, ok := token.Claims.(*tokenClaims); ok && token.Valid {
		if claims.ExpiresAt < time.Now().Unix() {
			return 0, fmt.Errorf("Token expired")
		}
		return claims.UserId, nil
	}
	return 0, err

}

func (s *UserService) GetUser(id int64) (*model.User, error) {
	return s.repo.GetUser(id)
}

func (s *UserService) UpdateStatus(u *model.User) error {
	return s.repo.UpdateStatus(u)
}
