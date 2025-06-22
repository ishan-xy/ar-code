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

	r.Post("/model", handlers.UploadModel, middleware.JWTProtected())
	r.Get("/model/files/:query", handlers.GetModelRedirect)
	r.Get("/qr/:query", handlers.GetModelQRCode)           // Direct PNG image
	r.Get("/api/qr/:query", handlers.GetModelQRCodeJSON)   // JSON with base64
	r.Get("/model/:query", handlers.GetModelMetadata, middleware.JWTProtected())
	r.Get("/model", handlers.GetAllModels, middleware.JWTProtected())
	r.Put("/model/:query", handlers.UpdateModel, middleware.JWTProtected())
	r.Delete("/model/:query", handlers.DeleteModel, middleware.JWTProtected())
}