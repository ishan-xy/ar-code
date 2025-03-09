package handlers

import (
	"backend/config"
	"backend/database"
	"backend/utility"
	"context"
	"fmt"
	"log"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/v2/bson"
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
	key, err := utility.HashFileSHA256(f)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	log.Println(key)

	// Reset file cursor to the beginning
	_, err = f.Seek(0, 0)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	m, exists, _ := database.AR_modelDB.GetExists(bson.M{"_id": key})
	if !exists {
		_, err = config.S3Client.PutObject(context.Background(), &s3.PutObjectInput{
			Bucket: &BucketName,
			Key:    &key,
			Body:   f,
		})

		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
		}
		cdnURL := fmt.Sprintf("https://cdn.%s/%s", config.Cfg.CdnDomain, key)
		query := utility.GenerateQuery(key)
		metadata := database.AR_model{
			ID:         key,
			FileName:   file.Filename,
			URL:        cdnURL,
			Query:      query,
			UploadDate: time.Now(),
		}
		_, err = database.AR_modelDB.InsertOne(context.Background(), metadata)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Model uploaded successfully", "metadata": m})
}
