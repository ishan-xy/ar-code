package middleware

import (
	"backend/utility"
	"backend/database"
	_ "log"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/gofiber/fiber/v3"
)



func ValidateAuthRequest() fiber.Handler{
	return func(c fiber.Ctx) error {
		var req database.AuthRequest
		if err := c.Bind().Body(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   utils.WithStack(err),
				"message": "Invalid request body",
			})
		}

		if req.Email == "" || req.Password == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"message": "Email and password are required"})
		}
		req.Email = utility.NormalizeEmail(req.Email)
		c.Locals("auth_request", req)
		return c.Next()
	}
}