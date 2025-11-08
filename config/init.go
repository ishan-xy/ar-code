package config

import (
	"context"
	"log"
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

	return &Config{
		Port:          Getenv("PORT"),
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

		FrontendURL:     "http://172.31.35.109:5511",
	}, nil
}
