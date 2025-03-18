package service

import (
	"GoSpeak/internal/model"
	"GoSpeak/internal/repository"
)

type MessageService struct {
	repo repository.Message
}

func NewMessageService(repo repository.Message) *MessageService {
	return &MessageService{
		repo: repo,
	}
}
func (s *MessageService) Send(m *model.Message) error {
	return s.repo.Send(m)
}
