package service

import (
	"GoSpeak/internal/model"
	"GoSpeak/internal/repository"
)

type ParticipantService struct {
	repo repository.Participant
}

func NewParticipantService(repo repository.Participant) *ParticipantService {
	return &ParticipantService{
		repo: repo,
	}
}
func (s *ParticipantService) AddToConference(u int64, conf *model.Conference) error {
	return s.repo.AddToConference(u, conf)
}
