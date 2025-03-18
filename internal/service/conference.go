package service

import (
	"GoSpeak/internal/model"
	"GoSpeak/internal/repository"

	"github.com/google/uuid"
)

type ConferenceService struct {
	repo repository.Conference
}

func NewConferenceService(repo repository.Conference) *ConferenceService {
	return &ConferenceService{repo: repo}
}
func (s *ConferenceService) CreateConference(c *model.Conference) (*model.Conference, error) {
	join_url := uuid.NewString()
	c.JoinURL = join_url

	err := s.repo.CreateConference(c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *ConferenceService) GetConference(join_url string) (*model.Conference, error) {
	return s.repo.GetConference(join_url)
}
