package service

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/storyvows/backend/integrations"
)

func NewS3Client(cfg *integrations.Secrets) (*s3.Client, error) {
	loadOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.S3Region),
	}

	if cfg.S3AccessKeyID != "" || cfg.S3SecretAccessKey != "" {
		if cfg.S3AccessKeyID == "" || cfg.S3SecretAccessKey == "" {
			return nil, fmt.Errorf("S3 credentials are incomplete: both S3_ACCESS_KEY_ID and S3_SECRET_ACCESS_KEY are required")
		}
		loadOptions = append(loadOptions,
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, ""),
			),
		)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), loadOptions...)
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
