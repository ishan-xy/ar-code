package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"backend/db"
	"backend/models"

	"github.com/gin-gonic/gin"
)

var modelDir = "./uploaded_models"

func UploadModelHandler(c *gin.Context) {
	name := c.PostForm("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name is required"})
		return
	}

	// Ensure folder structure for model files
	folderPath := filepath.Join(modelDir, name)
	counter := 1
	for {
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			break
		}
		folderPath = filepath.Join(modelDir, fmt.Sprintf("%s_%d", name, counter))
		counter++
	}

	if err := os.MkdirAll(folderPath, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create folder for model"})
		return
	}

	// Process GLB file (required)
	glbFile, err := c.FormFile("glb")
	if glbFile == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "GLB file is required"})
		return
	}

	glbPath := filepath.Join(folderPath, glbFile.Filename)
	if err := c.SaveUploadedFile(glbFile, glbPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save GLB file"})
		return
	}

	// Process USDZ file (optional)
	usdzFile, _ := c.FormFile("usdz")
	var usdzPath string
	if usdzFile != nil {
		usdzPath = filepath.Join(folderPath, usdzFile.Filename)
		if err := c.SaveUploadedFile(usdzFile, usdzPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save USDZ file"})
			return
		}
	}

	// Save to database
	entry := models.ModelEntry{
		ID:    fmt.Sprintf("%d", time.Now().UnixNano()),
		Name:  filepath.Base(folderPath),
		USDZ:  usdzPath,
		GLB:   glbPath,
	}

	collection := db.Client.Database("3dmodels").Collection("models")
	if _, err := collection.InsertOne(context.TODO(), entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save model entry to database"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Model uploaded successfully", "data": entry})
}
