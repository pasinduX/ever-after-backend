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
	"image/heif":      dto.FileTypePhoto,
	"image/avif":      dto.FileTypePhoto,
	"video/mp4":       dto.FileTypeVideo,
	"video/mov":       dto.FileTypeVideo,
	"video/quicktime": dto.FileTypeVideo,
}

type UploadService struct {
	db       *mongo.Database
	cfg      *integrations.Secrets
	s3       *s3.Client
	analysis *AnalysisService
}

func NewUploadService(db *mongo.Database, cfg *integrations.Secrets, s3Client *s3.Client, analysis *AnalysisService) (*UploadService, error) {
	return &UploadService{db: db, cfg: cfg, s3: s3Client, analysis: analysis}, nil
}

func (s *UploadService) buildFileURL(fileKey string) string {
	if s.cfg.S3PublicBaseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.S3PublicBaseURL, "/"), fileKey)
	}
	if s.cfg.S3Endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimRight(s.cfg.S3Endpoint, "/"), s.cfg.S3Bucket, fileKey)
	}
	if s.cfg.S3Region == "us-east-1" {
		return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.cfg.S3Bucket, fileKey)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.cfg.S3Bucket, s.cfg.S3Region, fileKey)
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

	fileURL := s.buildFileURL(fileKey)
	upload := &dto.Upload{
		ID:             uuid.NewString(),
		WeddingID:      weddingID,
		FileURL:        fileURL,
		FileKey:        fileKey,
		FileType:       fileType,
		MimeType:       mimeType,
		SizeBytes:      header.Size,
		Category:       dto.CategoryOther,
		AnalysisStatus: dto.AnalysisStatusPending,
		QualityScore:   nil,
		DetectedFaces:  nil,
		Orientation:    nil,
		SceneTags:      nil,
		AnalysisError:  nil,
		AIInsights:     nil,
		IsApproved:     true,
		UploadedAt:     time.Now(),
		Storage: dto.UploadStorage{
			OriginalURL:  fileURL,
			MediumURL:    fileURL,
			ThumbnailURL: fileURL,
			FileKey:      fileKey,
		},
		Metadata: dto.UploadMetadata{
			MimeType:  mimeType,
			SizeBytes: header.Size,
		},
		Timeline: dto.UploadTimeline{
			UploadedAt: time.Now(),
		},
		Analysis: dto.UploadAnalysis{
			Status:   dto.AnalysisStatusPending,
			Category: dto.CategoryOther,
			Processing: dto.ProcessingStages{
				Thumbnail:      dto.AnalysisStatusPending,
				AIAnalysis:     dto.AnalysisStatusPending,
				Moderation:     dto.AnalysisStatusPending,
				DuplicateCheck: dto.AnalysisStatusPending,
			},
		},
		Moderation: dto.UploadModeration{
			IsApproved: true,
		},
	}
	if guestName != "" {
		upload.GuestName = &guestName
	}

	if err := dao.CreateUpload(ctx, s.db, upload); err != nil {
		return nil, err
	}
	if s.analysis != nil {
		s.analysis.Enqueue(upload.ID)
	}
	return upload, nil
}

func (s *UploadService) GuestUploadBySlug(ctx context.Context, slug string, file multipart.File, header *multipart.FileHeader, guestName string) (*dto.Upload, error) {
	wedding, err := dao.FindWeddingBySlug(ctx, s.db, slug)
	if errors.Is(err, dao.ErrNoRows) {
		return nil, errors.New("wedding not found")
	}
	if err != nil {
		return nil, err
	}
	return s.GuestUpload(ctx, wedding.ID, file, header, guestName)
}

func (s *UploadService) GuestUploadByIdentifier(ctx context.Context, identifier string, file multipart.File, header *multipart.FileHeader, guestName string) (*dto.Upload, error) {
	wedding, err := dao.FindWeddingByID(ctx, s.db, identifier)
	if errors.Is(err, dao.ErrNoRows) {
		wedding, err = dao.FindWeddingBySlug(ctx, s.db, identifier)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	return s.GuestUpload(ctx, wedding.ID, file, header, guestName)
}

func (s *UploadService) UploadToFolder(ctx context.Context, folderID string, file multipart.File, header *multipart.FileHeader) (*dto.Upload, error) {
	if s.cfg.S3Bucket == "" {
		return nil, errors.New("S3 bucket is not configured")
	}

	mimeType := header.Header.Get("Content-Type")
	fileType, ok := allowedMimeTypes[mimeType]
	if !ok {
		return nil, apperrors.ErrInvalidFile
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	fileKey := fmt.Sprintf("%s/%s%s", strings.Trim(folderID, "/"), uuid.NewString(), ext)

	_, err := s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.S3Bucket),
		Key:         aws.String(fileKey),
		Body:        file,
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	fileURL := s.buildFileURL(fileKey)
	upload := &dto.Upload{
		ID:             uuid.NewString(),
		WeddingID:      folderID,
		FileURL:        fileURL,
		FileKey:        fileKey,
		FileType:       fileType,
		MimeType:       mimeType,
		SizeBytes:      header.Size,
		Category:       dto.CategoryOther,
		AnalysisStatus: dto.AnalysisStatusPending,
		QualityScore:   nil,
		DetectedFaces:  nil,
		Orientation:    nil,
		SceneTags:      nil,
		AnalysisError:  nil,
		AIInsights:     nil,
		IsApproved:     true,
		UploadedAt:     time.Now(),
		Storage: dto.UploadStorage{
			OriginalURL:  fileURL,
			MediumURL:    fileURL,
			ThumbnailURL: fileURL,
			FileKey:      fileKey,
		},
		Metadata: dto.UploadMetadata{
			MimeType:  mimeType,
			SizeBytes: header.Size,
		},
		Timeline: dto.UploadTimeline{
			UploadedAt: time.Now(),
		},
		Analysis: dto.UploadAnalysis{
			Status:   dto.AnalysisStatusPending,
			Category: dto.CategoryOther,
			Processing: dto.ProcessingStages{
				Thumbnail:      dto.AnalysisStatusPending,
				AIAnalysis:     dto.AnalysisStatusPending,
				Moderation:     dto.AnalysisStatusPending,
				DuplicateCheck: dto.AnalysisStatusPending,
			},
		},
		Moderation: dto.UploadModeration{
			IsApproved: true,
		},
	}

	if err := dao.CreateUpload(ctx, s.db, upload); err != nil {
		return nil, err
	}
	if s.analysis != nil {
		s.analysis.Enqueue(upload.ID)
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
