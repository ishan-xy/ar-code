package handlers

import (
	"backend/config"
	"backend/database"
	"context"
	"fmt"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v3"
)

var BucketName string = "ar-models"

func CreateModel(c fiber.Ctx) error {
	file, err := c.FormFile("model")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	defer f.Close()
	key := fmt.Sprintf("models/%s", file.Filename)

	_, err = config.S3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: &BucketName,
		Key:    &key,
		Body:   f,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	cdnURL := fmt.Sprintf("https://cdn.%s/%s", config.Cfg.CdnDomain, key)
	metadata := database.AR_model{
		ID:         key,
		FileName:   file.Filename,
		URL:        cdnURL,
		UploadDate: time.Now(),
	}
	_, err = database.AR_modelDB.InsertOne(context.Background(), metadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Model uploaded successfully"})
}
