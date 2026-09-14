package config

import (
	"context"
	"log"
	"os"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/go-redis/redis/v8"
)

type Config struct {
	Port          string
	MongoURI      string
	DBName        string
	Secret        string
	JWTExpiration time.Duration
	CookieName    string

	AccountID       string
	TokenValue      string
	AccessKeyID     string
	SecretAccessKey string
	CdnDomain       string
	BucketName      string

	FrontendURL     string
}

var Cfg *Config
var RedisClient *redis.Client
func init() {
	loadEnv()
	var err error
	Cfg, err = loadConfig()
	if err != nil {
		log.Fatal(utils.WithStack(err))
	}
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     Getenv("REDIS_ADDR"),
		Password: Getenv("REDIS_PASSWORD"),
		DB:       0, // use default DB
	})
	ping, err := RedisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", utils.WithStack(err))
	}
	log.Println("Connected to Redis:", ping)
	log.Println("Configuration loaded successfully:", Cfg)
}

func loadConfig() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	viewerURL := os.Getenv("VIEWER_URL")
	if viewerURL == "" {
		viewerURL = os.Getenv("FRONTEND_URL")
	}
	if viewerURL == "" {
		viewerURL = "https://v.gamchngr.xyz"
	}

	bucketName := os.Getenv("R2_BUCKET_NAME")
	if bucketName == "" {
		bucketName = os.Getenv("BUCKET_NAME")
	}
	if bucketName == "" {
		bucketName = "ar-models"
	}

	return &Config{
		Port:          port,
		MongoURI:      Getenv("MONGO_URI"),
		DBName:        Getenv("DBName"),
		Secret:        Getenv("SECRET"),
		JWTExpiration: time.Hour * 24,
		CookieName:    "sessionID",

		AccountID:       Getenv("AccountID"),
		TokenValue:      Getenv("TokenValue"),
		AccessKeyID:     Getenv("AccessKeyID"),
		SecretAccessKey: Getenv("SecretAccessKey"),
		CdnDomain:       Getenv("CDN_DOMAIN"),
		BucketName:      bucketName,

		FrontendURL:     viewerURL,
	}, nil
}
