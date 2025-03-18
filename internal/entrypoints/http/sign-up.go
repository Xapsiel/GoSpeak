package http

import (
	"GoSpeak/internal/model"

	"github.com/gofiber/fiber/v2"
)

type SignUpResponse struct {
	*Page
	*model.User
}
type SignUpPostResponse struct {
	Error string `json:"error"`
}

func (r *Router) PostSignUpHandler(c *fiber.Ctx) error {
	var u model.SignUpUser
	if err := c.BodyParser(&u); err != nil {
		return c.JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	_, err := r.service.User.SignUp(&u)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(SignUpPostResponse{})
}
func (r *Router) GetSignUpHandler(c *fiber.Ctx) error {
	resp := &SignUpResponse{
		Page: r.NewPage(),
	}
	return c.Render("sign-up", resp, "layouts/main")

}
