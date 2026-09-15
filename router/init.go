package router

import (
	"backend/config"
	"backend/handlers"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	fiberRecover "github.com/gofiber/fiber/v3/middleware/recover"
)

func init() {
	defer func() {
		if err := recover(); err != nil {
			log.Fatal(utils.WithStack(errors.New("Error initializing router: " + fmt.Sprint(err))))
		}
	}()
	app := fiber.New(fiber.Config{
		CaseSensitive:      true,
		Concurrency:        1024 * 1024,
		IdleTimeout:        30 * time.Second,
		DisableDefaultDate: true,
		JSONEncoder:        json.Marshal,
		JSONDecoder:        json.Unmarshal,
		BodyLimit:          100 * 1024 * 1024,
	})

	corsOrigins := os.Getenv("CORS_ORIGINS")
	var origins []string
	if corsOrigins != "" {
		for _, o := range strings.Split(corsOrigins, ",") {
			o = strings.TrimSpace(o)
			if o != "" {
				origins = append(origins, o)
			}
		}
	} else {
		origins = []string{
			"http://localhost:3000",
			"http://127.0.0.1:3000",
			"https://ar.gamchngr.xyz",
			"https://dev-ar.gamchngr.xyz",
			"https://v.gamchngr.xyz",
			"https://dev-v.gamchngr.xyz",
		}
	}

	app.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "HEAD", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}))

	app.Use(fiberRecover.New(fiberRecover.Config{EnableStackTrace: true}))
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} ${method} ${path} - ${latency} ${error}\n",
		TimeFormat: "2006-01-02 15:04:05",
		TimeZone:   "UTC",
	}))

	utils.SetErrorStackTrace(true)

	// Root and health check routes
	app.Get("/", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "online",
			"service": "ar-code-api",
		})
	})
	app.Get("/health", func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	authRoutes(app)
	modelRoutes(app)

	go func() {
		handlers.CleanupExpiredGuestModels()
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			handlers.CleanupExpiredGuestModels()
		}
	}()

	// Start the server bound to 0.0.0.0 for container and external proxy reachability
	port := config.Cfg.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("Server listening on 0.0.0.0:%s\n", port)
	log.Fatal(
		app.Listen("0.0.0.0:"+port, fiber.ListenConfig{
			EnablePrintRoutes: false,
		}),
	)
}