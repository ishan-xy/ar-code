package handlers

import (
	"backend/config"
	"backend/database"
	"backend/utility"
	"context"
	_ "fmt"
	"log"
	_ "log"
	"strings"
	"text/template"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gofiber/fiber/v3"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var BucketName string = "ar-models"

/*
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
			// cdnURL := fmt.Sprintf("https://cdn.%s/%s", config.Cfg.CdnDomain, key)
			presignedUrl, err := utility.GenerateR2PresignedURL(config.S3Client, BucketName, key)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
			}
			query := utility.GenerateQuery(key)
			log.Println(query)
			metadata := database.AR_model{
				ID:         key,
				FileName:   file.Filename,
				URL:        presignedUrl,
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

func ViewModel(c fiber.Ctx) error {
	// Get query parameter from URL
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing model query parameter",
		})
	}

	// Find model in database
	var model database.AR_model
	err := database.AR_modelDB.FindOne(context.Background(), bson.M{"query": query}).Decode(&model)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Model not found",
		})
	}

	// Render AR template with model URL
	c.Set("Content-Type", "text/html")
	tmpl := template.Must(template.ParseFiles("/home/ishan/dev/ar-code/handlers/ar.html"))
	return tmpl.Execute(c.Response().BodyWriter(), fiber.Map{
		"ModelURL": model.URL,
	})
}
*/

func UploadModel(c fiber.Ctx) error {
	// upload the model file directly, if a file with the same name exists, new file will be renamed
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
	// normalize and make the filename unique
	file.Filename, err = utility.NormalizeFileName(file.Filename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}

	_, err = config.S3Client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: &BucketName,
		Key:    &file.Filename,
		Body:   f,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	presignedUrl, err := utility.GenerateR2PresignedURL(config.S3Client, BucketName, file.Filename)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	metadata := database.AR_model{
		ID:            primitive.NewObjectID().Hex(),
		FileName:      file.Filename,
		URL:           presignedUrl,
		Query:         utility.GenerateQuery(file.Filename),
		UploadDate:    time.Now(),
		FileExtension: file.Filename[strings.LastIndex(file.Filename, ".")+1:],
	}
	_, err = database.AR_modelDB.InsertOne(context.Background(), metadata)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Model uploaded successfully", "metadata": metadata})

}

func GetModel(c fiber.Ctx) error {
	// Get query parameter from URL
	query := c.Query("q")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing model query parameter",
		})
	}

	// Find model in database
	m, exists, err := database.AR_modelDB.GetExists(bson.M{"query": query})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Error retrieving model",
		})
	}
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Model not found",
		})
	}
	// presignedUrl, err := utility.GenerateR2PresignedURL(config.S3Client, BucketName, m.FileName)
	presignedUrl := m.URL
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": utils.WithStack(err)})
	}
	if presignedUrl == "" {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Model URL not found",
		})
	}
	// Return the presigned URL
	//Renaming Issue Yet to fix
	c.Set("Content-Type", "application/json")
	log.Println("Presigned URL:", presignedUrl)
	return c.JSON(fiber.Map{
		"modelUrl": "https://ishanx.tech/carno.glb",
	})
}

func TestTemplate(c fiber.Ctx) error {
	// Set the Content-Type header to text/html
	c.Set("Content-Type", "text/html")

	tmpl := template.Must(template.ParseFiles("/home/ishan/dev/ar-code/handlers/index.html"))
	return tmpl.Execute(c.Response().BodyWriter(), nil)
}
