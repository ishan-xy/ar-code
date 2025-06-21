package utility

import (
	"context"
	"fmt"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func GenerateR2PresignedURL(s3Client *s3.Client, bucket, key string) (string, error) {
	presignClient := s3.NewPresignClient(s3Client)

	req, err := presignClient.PresignGetObject(context.Background(),
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
		s3.WithPresignExpires(24*time.Hour),
	)
	if err != nil {
		return "", utils.WithStack(fmt.Errorf("failed to presign R2 request: %w", err))
	}

	return req.URL, nil
}
