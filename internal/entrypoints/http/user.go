package http

import (
	"github.com/gofiber/fiber/v2"
)

func (r *Router) GetUserHandler(c *fiber.Ctx) error {
	tokenStr := c.Get("Authorization")[7:]
	if tokenStr == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Missing token")
	}

	u, err := r.service.User.ParseJWT(tokenStr)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
	}
	user, err := r.service.User.GetUser(u)
	r.service.User.UpdateStatus(user)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
	}

	return c.JSON(user)
}
