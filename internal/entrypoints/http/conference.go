package http

import (
	"log/slog"

	"GoSpeak/internal/model"

	"github.com/gofiber/fiber/v2"
)

type ConferenceResponse struct {
	*Page
	Description string
}

func (r *Router) JoinConferenceHandler(c *fiber.Ctx) error {

	joinUrl := c.Query("join_url")
	conf, err := r.service.Conference.GetConference(joinUrl)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Неверный формат данных")

	}
	u := c.Locals("user_id").(int64)

	//r.closeExistingConnections(u)

	err = r.service.Participant.AddToConference(u, conf.ConferenceID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Неверный формат данных")
	}

	return c.JSON(fiber.Map{
		"conference_id":          conf.ConferenceID,
		"conference_description": conf.Description,
		"creater_id":             conf.CreatorID,
		"join_url":               joinUrl,
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
		"creater_id": conf.CreatorID,
	})
}

func (r *Router) RenderConference(c *fiber.Ctx) error {
	resp := ConferenceResponse{Page: r.NewPage()}
	resp.Page.Title = "Видеоконференция"
	return c.Render("conference", resp, "layouts/main")
}

func (r *Router) IsUserInConfMiddleware(c *fiber.Ctx) error {
	u := c.Locals("user_id").(int64)
	ids, err := r.service.IsUserInConf(u)
	if err != nil {
		return c.Next()
	}
	if len(ids) > 0 {
		r.clientlock.Lock()
		for _, id := range ids {
			if _, ok := r.clients[id]; !ok {
				continue
			}
			for c, v := range r.clients[id].conn {
				if v.UserID == u {
					err = c.Close()
					if err != nil {
						slog.Error(err.Error())

					}
					delete(r.clients[id].conn, c)
				}
			}
		}
		r.clientlock.Unlock()
	}
	return c.Next()
}
