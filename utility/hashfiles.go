package utility

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"

	utils "github.com/ItsMeSamey/go_utils"
)

func HashFileSHA256(file multipart.File) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", utils.WithStack(err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}