package service

import (
	"GoSpeak/internal/model"
	"GoSpeak/internal/repository"
)

type ParticipantService struct {
	repo repository.Participant
}

func (s *ParticipantService) GetConferenceParticipants(id string) ([]model.Participant, error) {
	//TODO implement me
	panic("implement me")
}

func (s *ParticipantService) RemoveFromConference(id int64) error {
	return s.repo.RemoveFromConference(id)
}

func NewParticipantService(repo repository.Participant) *ParticipantService {
	return &ParticipantService{
		repo: repo,
	}
}
func (s *ParticipantService) AddToConference(u int64, conf *model.Conference) error {
	return s.repo.AddToConference(u, conf)
}
