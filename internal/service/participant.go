package service

import (
	"GoSpeak/internal/repository"
)

type ParticipantService struct {
	repo repository.Participant
}

func (s *ParticipantService) GetParticipantsByConferenceID(id string) ([]int64, error) {
	return s.repo.GetParticipantsByConferenceID(id)

}
func (s *ParticipantService) IsUserInConf(u int64) ([]string, error) {
	return s.repo.IsUserInConf(u)
}

func (s *ParticipantService) RemoveFromConference(id int64) error {
	return s.repo.RemoveFromConference(id)
}

func NewParticipantService(repo repository.Participant) *ParticipantService {
	return &ParticipantService{
		repo: repo,
	}
}
func (s *ParticipantService) AddToConference(u int64, conf string) error {
	return s.repo.AddToConference(u, conf)
}
