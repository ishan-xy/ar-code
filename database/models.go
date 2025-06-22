package database

import (
	"context"
	"errors"
	"log"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Name     string             `json:"name" bson:"name"`
	Username string             `json:"username" bson:"username"`
	Email    string             `json:"email" bson:"email"`
	Password []byte             `json:"password" bson:"password"`
}

type AR_model struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	OwnerID       primitive.ObjectID `bson:"owner_id"`
	DisplayName   string             `json:"display_name" bson:"display_name"`
	FileName      string             `json:"filename" bson:"filename"`
	Query         string             `json:"query" bson:"query"`
	Online        bool               `json:"online" bson:"online"`
	UploadDate    time.Time          `json:"upload_date" bson:"upload_date"`
	FileExtension string             `json:"file_ext" bson:"file_ext"`
}

type Collection[T any] struct {
	*mongo.Collection
}

func (c *Collection[T]) GetExists(filter any) (out T, exists bool, err error) {
	result := c.FindOne(context.Background(), filter)
	err = result.Err()

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return out, false, nil
		}
		log.Printf("Error finding document: %v\n", err)
		return out, false, utils.WithStack(err)
	}

	if err := result.Decode(&out); err != nil {
		log.Printf("Error decoding document: %v\n", err)
		return out, false, utils.WithStack(err)
	}

	return out, true, nil
}
