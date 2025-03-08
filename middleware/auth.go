package middleware

import (
	"backend/config"
	"backend/utility"
	_ "log"

	_ "github.com/ItsMeSamey/go_utils"
	"github.com/gofiber/fiber/v3"
)

func JWTProtected() fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenString := c.Cookies(config.Cfg.CookieName)
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		token, err := utility.ValidateJWT(tokenString)
		if err != nil || !token.Valid {
			// log.Printf("Error validating token: %v", utils.WithStack(err))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized"})
		}

		c.Locals("user", token)
		return c.Next()
	}
}