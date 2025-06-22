package handlers

import (
	"backend/config"
	"backend/database"
	"encoding/base64"
	"fmt"
	"strconv"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/gofiber/fiber/v3"
	"github.com/skip2/go-qrcode"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func GetModelQRCode(c fiber.Ctx) error {
	query := c.Params("query")

	// Look up model to verify it exists
	model, found, err := database.AR_modelDB.GetExists(bson.M{"query": query})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Model not found"})
	}

	// Build the AR viewer URL - this is what the QR code will contain
	frontendURL := config.Cfg.FrontendURL
	arViewerURL := fmt.Sprintf("%s/%s", frontendURL, query) // Results in: https://yourdomain.com/_nshai95ye

	// Get QR code size from query parameter (default: 256)
	qrSize := 256
	if sizeStr := c.Query("size"); sizeStr != "" {
		if parsed, err := strconv.Atoi(sizeStr); err == nil {
			qrSize = parsed
		}
	}
	if qrSize > 1024 {
		qrSize = 1024
	}
	if qrSize < 64 {
		qrSize = 64
	}

	// Generate QR code for the AR viewer URL
	qrBytes, err := qrcode.Encode(arViewerURL, qrcode.Medium, qrSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	// Set response headers
	c.Set("Content-Type", "image/png")
	c.Set("Cache-Control", "public, max-age=3600") // Cache for 1 hour
	c.Set("Content-Disposition", fmt.Sprintf("inline; filename=\"%s-qr.png\"", model.DisplayName))

	return c.Send(qrBytes)
}

// GetModelQRCodeJSON returns QR code as base64 in JSON response
func GetModelQRCodeJSON(c fiber.Ctx) error {
	query := c.Params("query")

	// Look up model to verify it exists
	model, found, err := database.AR_modelDB.GetExists(bson.M{"query": query})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Model not found"})
	}

	// Build the AR viewer URL
	frontendURL := config.Cfg.FrontendURL
	arViewerURL := fmt.Sprintf("%s/%s", frontendURL, query)

	// Get QR code size from query parameter (default: 256)
	qrSize := 256
	if sizeStr := c.Query("size"); sizeStr != "" {
		if parsed, err := strconv.Atoi(sizeStr); err == nil {
			qrSize = parsed
		}
	}
	if qrSize > 1024 {
		qrSize = 1024
	}
	if qrSize < 64 {
		qrSize = 64
	}

	// Generate QR code
	qrBytes, err := qrcode.Encode(arViewerURL, qrcode.Medium, qrSize)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	// Convert to base64
	base64QR := base64.StdEncoding.EncodeToString(qrBytes)

	return c.JSON(fiber.Map{
		"qr_code": fmt.Sprintf("data:image/png;base64,%s", base64QR),
		"ar_url": arViewerURL,
		"model_name": model.DisplayName,
		"query": model.Query,
	})
}