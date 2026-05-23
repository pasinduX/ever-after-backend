package service

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/storyvows/backend/integrations"
)

func NewS3Client(cfg *integrations.Secrets) (*s3.Client, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, ""),
		),
	)
	if err != nil {
		return nil, err
	}

	if cfg.S3Endpoint != "" {
		return s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
			o.UsePathStyle = true
		}), nil
	}

	return s3.NewFromConfig(awsCfg), nil
}
