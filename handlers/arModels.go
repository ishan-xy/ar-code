package handlers

import (
	"backend/config"
	"backend/database"
	"backend/utility"
	"context"
	"path/filepath"
	"strings"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var BucketName string = "ar-models"

func UploadModel(c fiber.Ctx) error {
	// get the username of the logged-in user
	userToken, _ := c.Locals("user").(*jwt.Token)
	_, username, err := utility.GetClaimsFromToken(userToken)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	user, _, err := database.UserDB.GetExists(bson.M{"username": username})

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	// Open the file
	file, err := c.FormFile("model")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	defer f.Close()
	// Reset file cursor to the beginning
	_, err = f.Seek(0, 0)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	// normalize the original filename first
	normalizedFilename := utility.NormalizeFileName(file.Filename)

	// Generate unique filename based on username
	uniqueFilename, err := utility.GenerateUniqueFilename(config.S3Client, BucketName, username, normalizedFilename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	// Get display name from the post request
	displayName := c.FormValue("displayName")
	if displayName == "" {
		displayName = c.FormValue("display_name") // fallback for snake_case
	}
	if displayName == "" {
		// If no display name provided, use the original filename without extension
		displayName = strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename))
	}
	// Trim whitespace and validate
	displayName = strings.TrimSpace(displayName)
	_, exists, err := database.AR_modelDB.GetExists(bson.M{"display_name":displayName, "owner_id": user.ID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if exists{
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "An Object with the same name already exists"})
	}
	if displayName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Display name cannot be empty"})
	}
	
	// Use the unique filename for S3 upload
	_, err = config.S3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: &BucketName,
		Key:    &uniqueFilename,
		Body:   f,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	
	metadata := database.AR_model{
		ID:            primitive.NewObjectID(),
		OwnerID:       user.ID,
		FileName:      uniqueFilename,
		DisplayName:   displayName,
		Query:         utility.GenerateQuery(uniqueFilename),
		UploadDate:    time.Now(),
		FileExtension: uniqueFilename[strings.LastIndex(uniqueFilename, ".")+1:],
	}

	_, err = database.AR_modelDB.InsertOne(context.Background(), metadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Model uploaded successfully", "metadata": metadata})
}

func GetModelRedirect(c fiber.Ctx) error {
	query := c.Params("query")

	// Look up model metadata by query
	model, found, err := database.AR_modelDB.GetExists(bson.M{"query": query})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Model not found"})
	}

	// Generates a fresh presigned URL every time
	// CACHING REQUIRED HERE, REFRESH AFTER 12 HOURS
	presignedUrl, err := utility.GenerateR2PresignedURL(config.S3Client, BucketName, model.FileName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	return c.Redirect().Status(fiber.StatusTemporaryRedirect).To(presignedUrl)
}
