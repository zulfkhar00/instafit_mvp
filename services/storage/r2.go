// r2.go
package storage

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"

	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Service struct {
	client     *s3.Client
	bucketName string
}

// Ensure R2Service implements StorageService
var _ StorageService = (*R2Service)(nil)

// Constructor for R2Service
func NewR2Service() (*R2Service, error) {
	accountId := os.Getenv("R2_ACCOUNT_ID")
	accessKeyId := os.Getenv("R2_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucket := os.Getenv("R2_BUCKET_NAME")

	customTransport := http.DefaultTransport.(*http.Transport).Clone()
	customTransport.Proxy = nil

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")),
		config.WithRegion("auto"),
		config.WithHTTPClient(&http.Client{Transport: customTransport}),
		config.WithRequestChecksumCalculation(0),
		config.WithResponseChecksumValidation(0),
	)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountId))
	})

	return &R2Service{
		client:     client,
		bucketName: bucket,
	}, nil
}

// UploadBLOB uploads any binary data to Cloudflare R2.
func (r *R2Service) UploadBlob(ctx context.Context, data []byte, filename, contentType string) (string, error) {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})

	if err != nil {
		return "", err
	}

	publicBaseURL := os.Getenv("R2_PUBLIC_URL")
	url := fmt.Sprintf("%s/%s", publicBaseURL, filename)
	return url, nil
}

// DeleteBlob deletes a blob with a key from Cloudflare R2.
func (s *R2Service) DeleteBlob(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}
