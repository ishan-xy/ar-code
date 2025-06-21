package utility

import (
	"backend/database"
	"log"
	"strconv"
	"strings"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"go.mongodb.org/mongo-driver/bson"
)

func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func GenerateQuery(modelID string) string {
	hashPrefix := modelID[:4]
	now := time.Now().UnixNano()
	timeComponent := now % (36 * 36)
	timeStr := strconv.FormatInt(timeComponent, 36)
	if len(timeStr) < 2 {
		timeStr = "0" + timeStr
	}
	log.Println(hashPrefix)
	return hashPrefix + timeStr
}

func NormalizeFileName(fileName string) (string, error) {
	// Remove leading and trailing spaces
	fileName = strings.TrimSpace(fileName)

	// Replace spaces with underscores
	fileName = strings.ReplaceAll(fileName, " ", "_")

	// Convert to lowercase
	fileName = strings.ToLower(fileName)

	// Ensure the file name is not empty
	if fileName == "" {
		fileName = "default_file_name"
	}

	// if a file with a same name exists in the database, increment a counter
	_, exists, err := database.AR_modelDB.GetExists(bson.M{"filename": fileName})
	if exists {
		counter := 1
		for exists {
			newFileName := fileName + "_" + strconv.Itoa(counter)
			_, exists, err = database.AR_modelDB.GetExists(bson.M{"filename": newFileName})
			if err != nil {
				log.Printf("Error checking for existing file name: %v\n", err)
				return "", utils.WithStack(err)
			}
			if !exists {
				fileName = newFileName
				break
			}
			counter++
		}
	}

	return fileName, utils.WithStack(err)
}
