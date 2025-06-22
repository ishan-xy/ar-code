package utility

import (
	"backend/config"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (hash []byte, err error) {
	hash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return hash, utils.WithStack(err)
}

func CheckPasswordHash(password string, hash []byte) bool {
	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	return err == nil
}

func GenerateJWT(email string, username string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["email"] = email
	claims["username"] = username
	claims["exp"] = time.Now().Add(config.Cfg.JWTExpiration).Unix()

	tokenString, err := token.SignedString([]byte(config.Cfg.Secret))
	if err != nil {
		return "", utils.WithStack(err)
	}

	return tokenString, nil
}

func ValidateJWT(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		
		// Validate that all required claims are present
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, fmt.Errorf("invalid token claims")
		}
		
		// Check if required claims exist
		if claims["email"] == nil || claims["username"] == nil || claims["exp"] == nil {
			return nil, fmt.Errorf("missing required claims")
		}
		
		// Validate email and username are non-empty strings
		email, ok := claims["email"].(string)
		if !ok || email == "" {
			return nil, fmt.Errorf("invalid email claim")
		}
		
		username, ok := claims["username"].(string)
		if !ok || username == "" {
			return nil, fmt.Errorf("invalid username claim")
		}
		
		return []byte(config.Cfg.Secret), nil
	})
}

// GetClaimsFromToken extracts claims from a validated JWT token
func GetClaimsFromToken(token *jwt.Token) (email, username string, err error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", fmt.Errorf("invalid token claims")
	}
	
	email, ok = claims["email"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid email claim")
	}
	
	username, ok = claims["username"].(string)
	if !ok {
		return "", "", fmt.Errorf("invalid username claim")
	}
	
	return email, username, nil
}

// IsValidFileExtension checks if the file extension is allowed for 3D models
func IsValidFileExtension(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	validExtensions := map[string]bool{
		".usdz": true,
		".glb":  true,
		".gltf": true,
		".obj":  true,
		".fbx":  true,
		".dae":  true,
		".3ds":  true,
		".ply":  true,
		".stl":  true,
	}
	
	return validExtensions[ext]
}

// GetFileSize returns human-readable file size
func GetFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}