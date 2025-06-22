package utility

import (
	"backend/database"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}


func GenerateQuery(modelID string) string {
	// Ensure modelID has at least 6 characters
	prefix := modelID
	if len(modelID) < 6 {
		prefix += strings.Repeat("x", 6-len(modelID))
	} else {
		prefix = modelID[:6]
	}

	// Create a local random generator
	src := rand.NewSource(time.Now().UnixNano())
	r := rand.New(src)

	// Shuffle the prefix characters
	prefixRunes := []rune(prefix)
	r.Shuffle(len(prefixRunes), func(i, j int) {
		prefixRunes[i], prefixRunes[j] = prefixRunes[j], prefixRunes[i]
	})
	shuffledPrefix := string(prefixRunes)

	// Generate a 4-character random base36 string
	randomComponent := r.Int63n(36 * 36 * 36 * 36)
	randomStr := strconv.FormatInt(randomComponent, 36)
	randomStr = fmt.Sprintf("%04s", randomStr) // pad to 4 chars if needed

	final := shuffledPrefix + strings.ToLower(randomStr)
	_, exists, err := database.AR_modelDB.GetExists(bson.M{"query":final})
	if err!=nil{
		log.Println(utils.WithStack(err))
	}
	if exists{
		return GenerateQuery(modelID)
	}
	return final // 10-character result
}


func NormalizeFileName(fileName string) (string) {
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

	return fileName
}

// Generate unique filename based on username and original filename
func GenerateUniqueFilename(s3Client *s3.Client, bucket, username, originalFilename string) (string, error) {
	// Clean the original filename to remove any path components
	originalFilename = filepath.Base(originalFilename)

	// Extract file extension
	ext := filepath.Ext(originalFilename)
	nameWithoutExt := strings.TrimSuffix(originalFilename, ext)

	// Generate base filename: username_originalname.ext
	baseFilename := fmt.Sprintf("%s_%s%s", username, nameWithoutExt, ext)

	// Check if base filename exists
	exists, err := ObjectExists(s3Client, bucket, baseFilename)
	if err != nil {
		return "", fmt.Errorf("error checking if file exists: %w", err)
	}

	// If base filename doesn't exist, use it
	if !exists {
		return baseFilename, nil
	}

	// If it exists, find the next available version number
	counter := 1
	for {
		versionedFilename := fmt.Sprintf("%s_%s_%d%s", username, nameWithoutExt, counter, ext)

		exists, err := ObjectExists(s3Client, bucket, versionedFilename)
		if err != nil {
			return "", fmt.Errorf("error checking if versioned file exists: %w", err)
		}

		if !exists {
			return versionedFilename, nil
		}

		counter++

		// Safety check to prevent infinite loop
		if counter > 10000 {
			return "", fmt.Errorf("too many versions of file %s for user %s", originalFilename, username)
		}
	}
}