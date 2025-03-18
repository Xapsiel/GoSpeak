package http

import (
	"GoSpeak/internal/model"

	"github.com/gofiber/fiber/v2"
)

type JoinConferenceResponse struct {
	*Page
	*model.User
	*model.Conference
}

func (r *Router) JoinConferenceHandler(c *fiber.Ctx) error {

	join_url := c.Query("join_url")
	conf, err := r.service.Conference.GetConference(join_url)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Неверный формат данных")

	}
	u := c.Locals("user_id").(int64)
	err = r.service.Participant.AddToConference(u, conf)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Неверный формат данных")
	}

	return c.JSON(fiber.Map{
		"conference_id": conf.ConferenceID,
		"creater_id":    conf.CreaterID,
		"join_url":      join_url,
	})

}

func (r *Router) CreateConferenceHandler(c *fiber.Ctx) error {
	var conf *model.Conference
	if err := c.BodyParser(&conf); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Неверный формат данных")
	}

	conf, err := r.service.Conference.CreateConference(conf)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"join_url":   conf.JoinURL,
		"creater_id": conf.CreaterID,
	})
}
