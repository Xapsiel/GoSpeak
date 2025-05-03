package http

import (
	"GoSpeak/internal/model"

	"github.com/gofiber/fiber/v2"
)

type SignInResponse struct {
	*Page
	*model.User
}

type SignInPostResponse struct {
	User     *model.User `json:"user"`
	JWTToken string      `json:"jwt_token"`

	Error string `json:"error"`
}

func (r *Router) PostSignInHandler(c *fiber.Ctx) error {
	var u model.SignUpUser
	if err := c.BodyParser(&u); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Неверный формат данных")
	}

	user, jwt, err := r.service.User.SignIn(&u)
	if err != nil {
		return c.JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	if user == nil {
		return c.JSON(fiber.Map{
			"error": "Пользователь не найден",
		})
	}

	c.Locals("jwt_token", jwt)

	return c.JSON(SignInPostResponse{
		User:     user,
		JWTToken: jwt,
	})
}

func (r *Router) RenderSignIn(c *fiber.Ctx) error {
	resp := &SignInResponse{
		Page: r.NewPage(),
	}
	resp.Page.Title = "Авторизация"
	return c.Render("sign-in", resp, "layouts/main")
}

func (r *Router) JWTMiddleware(c *fiber.Ctx) error {
	tokenStr := c.Get("Authorization")
	if len(tokenStr) > 7 {
		tokenStr = tokenStr[7:]
	} else {
		return fiber.NewError(fiber.StatusUnauthorized, "Missing token")

	}
	if tokenStr == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "Missing token")
	}

	u, err := r.service.User.ParseJWT(tokenStr) // Преобразуем в строку
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "Invalid token")
	}
	c.Locals("user_id", u)
	return c.Next()
}

func (r *Router) LogoutHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Успешный выход из системы",
	})
}
