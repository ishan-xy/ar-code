package middleware

import (
	"backend/config"
	"backend/utility"
	"strings" // Import the strings package

	"github.com/gofiber/fiber/v3"
)

func JWTProtected() fiber.Handler {
	return func(c fiber.Ctx) error {
		var tokenString string
		
		// 1. Check for token in the Authorization header
		authHeader := c.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// 2. Fallback to checking the cookie
			tokenString = c.Cookies(config.Cfg.CookieName)
		}

		// If no token is found in either location, return an error
		if tokenString == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing or malformed JWT"})
		}

		// Validate the token
		token, err := utility.ValidateJWT(tokenString)
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired JWT"})
		}
		
		// Set user context and proceed
		c.Locals("user", token)
		return c.Next()
	}
}