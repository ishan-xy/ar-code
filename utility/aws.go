package utility

import (
	"context"
	"errors"
	"fmt"
	"time"

	utils "github.com/ItsMeSamey/go_utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

func ObjectExists(s3Client *s3.Client, bucket, key string) (bool, error) {
	_, err := s3Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) {
			return false, nil
		}
		// Return other errors (permission denied, network issues, etc.)
		return false, err
	}
	
	return true, nil
}
