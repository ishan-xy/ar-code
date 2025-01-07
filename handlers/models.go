package handlers

import (
	"context"
	"net/http"

	"backend/db"
	"backend/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

func GetModelsHandler(c *gin.Context) {
	collection := db.Client.Database("3dmodels").Collection("models")

	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch models"})
		return
	}
	defer cursor.Close(context.TODO())

	var models []models.ModelEntry
	if err := cursor.All(context.TODO(), &models); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode model entries"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": models})
}
