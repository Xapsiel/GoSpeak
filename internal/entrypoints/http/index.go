package http

import (
	"GoSpeak/internal/model"

	"github.com/gofiber/fiber/v2"
)

type IndexResponse struct {
	*Page
	*model.User
}

func (r *Router) IndexHandler(c *fiber.Ctx) error {
	resp := IndexResponse{
		Page: r.NewPage(),
	}
	//if err == nil {
	//	resp.User = u
	//}

	resp.Page.Title = "index page"
	return c.Render("index", resp, "layouts/main")
}
