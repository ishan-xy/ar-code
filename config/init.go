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

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	RedisClient = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0, // use default DB
	})
	ping, err := RedisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", utils.WithStack(err))
	}
	log.Println("Connected to Redis:", ping)
	log.Println("Configuration loaded successfully")
}

func loadConfig() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	viewerURL := os.Getenv("VIEWER_URL")
	if viewerURL == "" {
		viewerURL = "https://v.gamchngr.xyz"
	}

	bucketName := os.Getenv("R2_BUCKET_NAME")
	if bucketName == "" {
		bucketName = "ar-models"
	}

	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = os.Getenv("DBName")
	}
	if dbName == "" {
		dbName = "ar-code-dev"
	}

	return &Config{
		Port:          port,
		MongoURI:      Getenv("MONGO_URI"),
		DBName:        dbName,
		Secret:        Getenv("SECRET"),
		JWTExpiration: time.Hour * 24,
		CookieName:    "sessionID",

		AccountID:       Getenv("CF_ACCOUNT_ID"),
		AccessKeyID:     Getenv("CF_ACCESS_KEY_ID"),
		SecretAccessKey: Getenv("CF_SECRET_ACCESS_KEY"),
		CdnDomain:       Getenv("CDN_DOMAIN"),
		BucketName:      bucketName,

		FrontendURL:     viewerURL,
	}, nil
}
