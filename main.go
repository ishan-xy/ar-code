package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ModelEntry struct {
	ID      string `bson:"_id" json:"id"`
	PageID  string `bson:"page_id" json:"page_id"`
	Name    string `bson:"name" json:"name"`
	USDZ    string `bson:"usdz" json:"usdz"`
	GLB     string `bson:"glb" json:"glb"`
}

var (
	dbClient     *mongo.Client
	databaseName = "3dmodels"
	collection   = "models"
	modelDir     = "./uploaded_models"
)

func generatePageID() (string, error) {
	bytes := make([]byte, 4) // 4 bytes will give us 8 hex characters
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found: %v", err)
	}

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	var err error
	dbClient, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(dbURL))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer dbClient.Disconnect(context.TODO())

	if err = os.MkdirAll(modelDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create model directory: %v", err)
	}

	r := gin.Default()
	
	r.Static("/uploads", "./uploaded_models")
	r.LoadHTMLGlob("templates/*")
	
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "3D Model Upload",
		})
	})

	r.GET("/view/:pageID", func(c *gin.Context) {
		pageID := c.Param("pageID")
		collection := dbClient.Database(databaseName).Collection(collection)
		
		var model ModelEntry
		err := collection.FindOne(context.TODO(), bson.M{"page_id": pageID}).Decode(&model)
		if err != nil {
			c.HTML(http.StatusNotFound, "404.html", gin.H{
				"message": "Model not found",
			})
			return
		}
		
		c.HTML(http.StatusOK, "view.html", gin.H{
			"model": model,
		})
	})
	
	r.POST("/upload", uploadModelHandler)
	r.GET("/models", getModelsHandler)
	r.Run(":8080")
}

func uploadModelHandler(c *gin.Context) {
    name := c.PostForm("name")
    log.Println("Name received:", name)
    if name == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
        return
    }

    glbFile, err := c.FormFile("glb")
    if glbFile == nil {
        log.Println("GLB file is missing")
        c.JSON(http.StatusBadRequest, gin.H{"error": "GLB file is required"})
        return
    }

    pageID, err := generatePageID()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate page ID"})
        return
    }

    folderPath := filepath.Join(modelDir, pageID)
    if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
        log.Printf("Failed to create directory: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
        return
    }
    log.Println("Directory created:", folderPath)

    glbPath := filepath.Join(folderPath, glbFile.Filename)
    log.Println("Saving GLB file to:", glbPath)
    if err = c.SaveUploadedFile(glbFile, glbPath); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save GLB file"})
        return
    }

    var usdzPath string
    usdzFile, err := c.FormFile("usdz")
    if usdzFile != nil {
        usdzPath = filepath.Join(folderPath, usdzFile.Filename)
        log.Println("Saving USDZ file to:", usdzPath)
        if err = c.SaveUploadedFile(usdzFile, usdzPath); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save USDZ file"})
            return
        }
    }

    entry := ModelEntry{
        ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
        PageID:  pageID,
        Name:    name,
        USDZ:    usdzPath,
        GLB:     glbPath,
    }
    log.Printf("Model entry: %+v\n", entry)

    collection := dbClient.Database(databaseName).Collection(collection)
    _, err = collection.InsertOne(context.TODO(), entry)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save model entry to database"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "message": "Model uploaded successfully", 
        "data": entry,
        "viewUrl": fmt.Sprintf("/view/%s", pageID),
    })
}


func getModelsHandler(c *gin.Context) {
	collection := dbClient.Database(databaseName).Collection(collection)

	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch models"})
		return
	}
	defer cursor.Close(context.TODO())

	var models []ModelEntry
	for cursor.Next(context.TODO()) {
		var model ModelEntry
		if err := cursor.Decode(&model); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode model entry"})
			return
		}
		models = append(models, model)
	}

	c.JSON(http.StatusOK, gin.H{"data": models})
}