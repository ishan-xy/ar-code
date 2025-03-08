package config

import (
	"log"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
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
}

var Cfg *Config

func init() {
	loadEnv()
	var err error
	Cfg, err = loadConfig()
	if err != nil {
		log.Fatal(utils.WithStack(err))
	}
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
	}, nil
}
