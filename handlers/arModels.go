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

type ModelReturnData struct {
	ID            primitive.ObjectID `json:"id"`
	DisplayName   string             `json:"display_name"`
	Query         string             `json:"query"`
	UploadDate    time.Time          `json:"upload_date"`
	FileExtension string             `json:"file_ext"`
	ModelURL      string             `json:"model_url"`
	QR_Code       string             `json:"qr_code"`
}

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
	_, exists, err := database.AR_modelDB.GetExists(bson.M{"display_name": displayName, "owner_id": user.ID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if exists {
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

func GetModelMetadata(c fiber.Ctx) error {
	query := c.Params("query")

	// Look up model metadata by query
	model, found, err := database.AR_modelDB.GetExists(bson.M{"query": query})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if !found {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Model not found"})
	}
	modelMeta := ModelReturnData{
		ID:            model.ID,
		DisplayName:   model.DisplayName,
		Query:         model.Query,
		UploadDate:    model.UploadDate,
		FileExtension: model.FileExtension,
		ModelURL:      "/model/files/" + model.Query,
		QR_Code:       "/qr/" + model.Query,
	}

	return c.Status(fiber.StatusOK).JSON(modelMeta)
}

func GetAllModels(c fiber.Ctx) error {
	// Extract user ID from JWT claims
	userToken := c.Locals("user").(*jwt.Token)
	claims := userToken.Claims.(jwt.MapClaims)
	user,_,err := database.UserDB.GetExists(bson.M{"username": claims["username"]})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	ownerID := user.ID

	// Query for models owned by the user
	cursor, err := database.AR_modelDB.Collection.Find(context.Background(), bson.M{"owner_id": ownerID})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	defer cursor.Close(context.Background())

	var modelList []ModelReturnData
	for cursor.Next(context.Background()) {
		var model database.AR_model
		if err := cursor.Decode(&model); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
		}

		modelMeta := ModelReturnData{
			ID:            model.ID,
			DisplayName:   model.DisplayName,
			Query:         model.Query,
			UploadDate:    model.UploadDate,
			FileExtension: model.FileExtension,
			ModelURL:      "/model/files/" + model.Query,
			QR_Code:       "/qr/" + model.Query,
		}
		modelList = append(modelList, modelMeta)
	}

	if err := cursor.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	return c.Status(fiber.StatusOK).JSON(modelList)
}

