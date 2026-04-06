package s3

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"bug-report-service/internal/infrastructure/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func NewClient(ctx context.Context, cfg config.Config) (*s3.Client, error) {
	return newClient(ctx, cfg, cfg.S3.Endpoint)
}

func NewPublicClient(ctx context.Context, cfg config.Config) (*s3.Client, error) {
	endpoint := cfg.S3.PublicEndpoint
	if strings.TrimSpace(endpoint) == "" {
		endpoint = cfg.S3.Endpoint
	}
	return newClient(ctx, cfg, endpoint)
}

func newClient(ctx context.Context, cfg config.Config, endpoint string) (*s3.Client, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, errors.New("S3_ENDPOINT is empty")
	}
	if strings.TrimSpace(cfg.S3.Region) == "" {
		return nil, errors.New("S3_REGION is empty")
	}
	if strings.TrimSpace(cfg.S3.AccessKey) == "" || strings.TrimSpace(cfg.S3.SecretKey) == "" {
		return nil, errors.New("S3_ACCESS_KEY or S3_SECRET_KEY is empty")
	}
	if _, err := url.Parse(endpoint); err != nil {
		return nil, err
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(cfg.S3.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.S3.AccessKey, cfg.S3.SecretKey, "")),
	)
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.BaseEndpoint = aws.String(endpoint)
	})
	return client, nil
}
