package http

import (
	"github.com/gofiber/fiber/v2"
)

type IndexResponse struct {
	*Page
}

func (r *Router) IndexHandler(c *fiber.Ctx) error {
	resp := IndexResponse{
		Page: r.NewPage(),
	}
	resp.Page.Title = "GoSpeak - твой сервис для видеоконференций"
	return c.Render("index", resp, "layouts/main")
}
