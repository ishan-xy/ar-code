package config

import (
	"context"
	"log"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var S3Client *s3.Client

func init() {
	// Load AWS config with R2-specific settings
	r2Cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("auto"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			Cfg.AccessKeyID,
			Cfg.SecretAccessKey,
			"",
		)),
		config.WithBaseEndpoint("https://" + Cfg.AccountID + ".r2.cloudflarestorage.com"),
	)
	if err != nil {
		log.Println(utils.WithStack(err))
		return
	}

	S3Client = s3.NewFromConfig(r2Cfg)
	log.Println("S3 client initialized successfully")
}