package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	apperrors "github.com/storyvows/backend/errors"
	"github.com/storyvows/backend/integrations"
	"go.mongodb.org/mongo-driver/mongo"
)

var allowedMimeTypes = map[string]dto.FileType{
	"image/jpeg":      dto.FileTypePhoto,
	"image/png":       dto.FileTypePhoto,
	"image/webp":      dto.FileTypePhoto,
	"image/heic":      dto.FileTypePhoto,
	"video/mp4":       dto.FileTypeVideo,
	"video/mov":       dto.FileTypeVideo,
	"video/quicktime": dto.FileTypeVideo,
}

type UploadService struct {
	db  *mongo.Database
	cfg *integrations.Secrets
	s3  *s3.Client
}

func NewUploadService(db *mongo.Database, cfg *integrations.Secrets) (*UploadService, error) {
	var s3Client *s3.Client
	if cfg.S3Endpoint != "" {
		awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
			awsconfig.WithRegion(cfg.S3Region),
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, ""),
			),
		)
		if err != nil {
			return nil, err
		}
		s3Client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
			o.UsePathStyle = true
		})
	} else {
		awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
			awsconfig.WithRegion(cfg.S3Region),
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, ""),
			),
		)
		if err != nil {
			return nil, err
		}
		s3Client = s3.NewFromConfig(awsCfg)
	}
	return &UploadService{db: db, cfg: cfg, s3: s3Client}, nil
}

func (s *UploadService) GuestUpload(ctx context.Context, weddingID string, file multipart.File, header *multipart.FileHeader, guestName string) (*dto.Upload, error) {
	mimeType := header.Header.Get("Content-Type")
	fileType, ok := allowedMimeTypes[mimeType]
	if !ok {
		return nil, apperrors.ErrInvalidFile
	}

	wedding, err := dao.FindWeddingByID(ctx, s.db, weddingID)
	if errors.Is(err, dao.ErrNoRows) {
		return nil, errors.New("wedding not found")
	}
	if err != nil {
		return nil, err
	}

	if limit := wedding.UploadLimit(); limit != -1 {
		count, _ := dao.CountUploadsByWedding(ctx, s.db, weddingID)
		if count >= limit {
			return nil, apperrors.ErrLimitReached
		}
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	fileKey := fmt.Sprintf("weddings/%s/%s%s", weddingID, uuid.NewString(), ext)

	_, err = s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.S3Bucket),
		Key:         aws.String(fileKey),
		Body:        file,
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	fileURL := fmt.Sprintf("%s/%s", s.cfg.S3PublicBaseURL, fileKey)
	upload := &dto.Upload{
		ID:         uuid.NewString(),
		WeddingID:  weddingID,
		FileURL:    fileURL,
		FileKey:    fileKey,
		FileType:   fileType,
		MimeType:   mimeType,
		SizeBytes:  header.Size,
		Category:   dto.CategoryOther,
		IsApproved: true,
		UploadedAt: time.Now(),
	}
	if guestName != "" {
		upload.GuestName = &guestName
	}

	if err := dao.CreateUpload(ctx, s.db, upload); err != nil {
		return nil, err
	}
	return upload, nil
}

func (s *UploadService) ListForWedding(ctx context.Context, weddingID string) ([]*dto.Upload, error) {
	return dao.FindUploadsByWedding(ctx, s.db, weddingID)
}

func (s *UploadService) SetApproval(ctx context.Context, uploadID string, approved bool) error {
	return dao.SetUploadApproval(ctx, s.db, uploadID, approved)
}

func (s *UploadService) Delete(ctx context.Context, uploadID string) error {
	upload, err := dao.FindUploadByID(ctx, s.db, uploadID)
	if err != nil {
		return err
	}
	_, _ = s.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.cfg.S3Bucket),
		Key:    aws.String(upload.FileKey),
	})
	return dao.DeleteUpload(ctx, s.db, uploadID)
}
