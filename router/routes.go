package router

import (
	"backend/handlers"
	"backend/middleware"

	"github.com/gofiber/fiber/v3"
)

func authRoutes(r fiber.Router) {
	r.Post("/signup", handlers.SignUp, middleware.ValidateAuthRequest())
	r.Post("/login", handlers.Login, middleware.ValidateAuthRequest())
	r.Get("/logout", handlers.Logout, middleware.JWTProtected())
}

func modelRoutes(r fiber.Router) {
	// r.Get("/models", handlers.GetModels, middleware.JWTProtected())
	// r.Get("/models/:id", handlers.GetModel, middleware.JWTProtected())
	r.Post("/models", handlers.CreateModel, middleware.JWTProtected())
	// r.Put("/models/:id", handlers.UpdateModel, middleware.JWTProtected())
	// r.Delete("/models/:id", handlers.DeleteModel, middleware.JWTProtected())
}