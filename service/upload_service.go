package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"mime/multipart"
	neturl "net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/corona10/goimagehash"
	"github.com/google/uuid"
	exiflib "github.com/rwcarlsen/goexif/exif"
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
	db        *mongo.Database
	cfg       *integrations.Secrets
	s3        *s3.Client
	presigner *s3.PresignClient
	analysis  *AnalysisService
}

func NewUploadService(db *mongo.Database, cfg *integrations.Secrets, s3Client *s3.Client, analysis *AnalysisService) (*UploadService, error) {
	presigner := s3.NewPresignClient(s3Client)
	return &UploadService{db: db, cfg: cfg, s3: s3Client, presigner: presigner, analysis: analysis}, nil
}

// FreshURL generates a new presigned GET URL for the given S3 file key.
// Call this at serve time so URLs are never stale when delivered to the client.
func (s *UploadService) FreshURL(fileKey string) string {
	if fileKey == "" {
		return ""
	}
	url, err := s.buildFileURL(fileKey)
	if err != nil {
		return ""
	}
	return url
}

func (s *UploadService) buildFileURL(fileKey string) (string, error) {
	if s.cfg.S3Bucket == "" {
		return "", errors.New("s3 bucket is not configured")
	}

	if s.cfg.S3PublicBaseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(s.cfg.S3PublicBaseURL, "/"), fileKey), nil
	}

	rawURL := s.buildRawFileURL(fileKey)
	if s.s3 == nil || s.presigner == nil {
		return rawURL, nil
	}

	req, err := s.presigner.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.cfg.S3Bucket),
		Key:    aws.String(fileKey),
	}, func(po *s3.PresignOptions) {
		po.Expires = s.cfg.S3PresignTTL
	})
	if err != nil {
		slog.Warn("failed to create presigned url, falling back to raw public url", "error", err.Error(), "file_key", fileKey)
		return rawURL, nil
	}
	return req.URL, nil
}

// RefreshURL re-signs a previously issued file URL using the S3 object key
// embedded in it, so callers never serve back an expired presigned GET URL.
// URLs that aren't ours to re-sign (public base URL, external links) are
// returned unchanged.
func (s *UploadService) RefreshURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	fileKey, ok := s.extractFileKey(rawURL)
	if !ok {
		return rawURL
	}
	fresh, err := s.buildFileURL(fileKey)
	if err != nil || fresh == "" {
		return rawURL
	}
	return fresh
}

// extractFileKey recovers the S3 object key from a URL previously produced by
// buildRawFileURL/PresignGetObject, so it can be re-signed at serve time.
func (s *UploadService) extractFileKey(rawURL string) (string, bool) {
	if s.cfg.S3Bucket == "" {
		return "", false
	}
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return "", false
	}

	if strings.HasPrefix(u.Host, s.cfg.S3Bucket+".s3.") && strings.HasSuffix(u.Host, ".amazonaws.com") {
		return strings.TrimPrefix(u.Path, "/"), true
	}

	if s.cfg.S3Endpoint != "" {
		if endpointHost, err := neturl.Parse(s.cfg.S3Endpoint); err == nil && endpointHost.Host == u.Host {
			prefix := "/" + s.cfg.S3Bucket + "/"
			if strings.HasPrefix(u.Path, prefix) {
				return strings.TrimPrefix(u.Path, prefix), true
			}
		}
	}

	return "", false
}

func (s *UploadService) buildRawFileURL(fileKey string) string {
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
		return nil, apperrors.ErrWeddingNotFound
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

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	exifData := extractEXIF(fileBytes)
	contentHash := computeContentHash(fileBytes)
	dims, _ := extractDimensions(fileBytes)
	pHash, _ := computePHash(fileBytes)

	ext := strings.ToLower(filepath.Ext(header.Filename))
	fileKey := fmt.Sprintf("weddings/%s/%s%s", weddingID, uuid.NewString(), ext)

	_, err = s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.S3Bucket),
		Key:         aws.String(fileKey),
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	fileURL, err := s.buildFileURL(fileKey)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	timeline := dto.UploadTimeline{UploadedAt: now}
	if exifData.CapturedAt != nil {
		timeline.CapturedAt = exifData.CapturedAt
	}

	upload := &dto.Upload{
		ID:             uuid.NewString(),
		WeddingID:      weddingID,
		FileURL:        fileURL,
		FileKey:        fileKey,
		FileType:       fileType,
		MimeType:       mimeType,
		SizeBytes:      header.Size,
		ContentHash:    contentHash,
		Category:       dto.CategoryOther,
		AnalysisStatus: dto.AnalysisStatusPending,
		IsApproved:     true,
		UploadedAt:     now,
		Storage: dto.UploadStorage{
			OriginalURL:  fileURL,
			MediumURL:    fileURL,
			ThumbnailURL: fileURL,
			FileKey:      fileKey,
		},
		Metadata: dto.UploadMetadata{
			MimeType:  mimeType,
			SizeBytes: header.Size,
			Width:       func() *int { if dims.Width > 0 { v := dims.Width; return &v }; return nil }(),
			Height:      func() *int { if dims.Height > 0 { v := dims.Height; return &v }; return nil }(),
			AspectRatio: func() *float64 { if dims.AspectRatio > 0 { v := dims.AspectRatio; return &v }; return nil }(),
			Orientation: func() *string { if dims.Orientation != "" { v := dims.Orientation; return &v }; return nil }(),
		},
		Timeline:    timeline,
		EXIF:        exifData,
		PHash:       pHash,
		Orientation: func() *string { if dims.Orientation != "" { v := dims.Orientation; return &v }; return nil }(),
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
	if exifData.CapturedAt != nil {
		upload.TakenAt = exifData.CapturedAt
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
		if errors.Is(err, dao.ErrNoRows) {
			return nil, apperrors.ErrWeddingNotFound
		}
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

	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	exifData := extractEXIF(fileBytes)
	contentHash := computeContentHash(fileBytes)
	dims, _ := extractDimensions(fileBytes)
	pHash, _ := computePHash(fileBytes)

	ext := strings.ToLower(filepath.Ext(header.Filename))
	fileKey := fmt.Sprintf("%s/%s%s", strings.Trim(folderID, "/"), uuid.NewString(), ext)

	_, err = s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.S3Bucket),
		Key:         aws.String(fileKey),
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String(mimeType),
	})
	if err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	fileURL, err := s.buildFileURL(fileKey)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	timeline := dto.UploadTimeline{UploadedAt: now}
	if exifData.CapturedAt != nil {
		timeline.CapturedAt = exifData.CapturedAt
	}

	upload := &dto.Upload{
		ID:             uuid.NewString(),
		WeddingID:      folderID,
		FileURL:        fileURL,
		FileKey:        fileKey,
		FileType:       fileType,
		MimeType:       mimeType,
		SizeBytes:      header.Size,
		ContentHash:    contentHash,
		Category:       dto.CategoryOther,
		AnalysisStatus: dto.AnalysisStatusPending,
		IsApproved:     true,
		UploadedAt:     now,
		Storage: dto.UploadStorage{
			OriginalURL:  fileURL,
			MediumURL:    fileURL,
			ThumbnailURL: fileURL,
			FileKey:      fileKey,
		},
		Metadata: dto.UploadMetadata{
			MimeType:    mimeType,
			SizeBytes:   header.Size,
			Width:       func() *int { if dims.Width > 0 { v := dims.Width; return &v }; return nil }(),
			Height:      func() *int { if dims.Height > 0 { v := dims.Height; return &v }; return nil }(),
			AspectRatio: func() *float64 { if dims.AspectRatio > 0 { v := dims.AspectRatio; return &v }; return nil }(),
			Orientation: func() *string { if dims.Orientation != "" { v := dims.Orientation; return &v }; return nil }(),
		},
		Timeline:    timeline,
		EXIF:        exifData,
		PHash:       pHash,
		Orientation: func() *string { if dims.Orientation != "" { v := dims.Orientation; return &v }; return nil }(),
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
	if exifData.CapturedAt != nil {
		upload.TakenAt = exifData.CapturedAt
	}

	if err := dao.CreateUpload(ctx, s.db, upload); err != nil {
		return nil, err
	}
	if s.analysis != nil {
		s.analysis.Enqueue(upload.ID)
	}

	return upload, nil
}

// --- file helpers ---

func extractEXIF(data []byte) dto.UploadEXIF {
	var result dto.UploadEXIF
	x, err := exiflib.Decode(bytes.NewReader(data))
	if err != nil {
		return result
	}
	if tag, err := x.Get(exiflib.Make); err == nil {
		result.CameraMake, _ = tag.StringVal()
	}
	if tag, err := x.Get(exiflib.Model); err == nil {
		result.CameraModel, _ = tag.StringVal()
	}
	if t, err := x.DateTime(); err == nil {
		result.CapturedAt = &t
	}
	if lat, lng, err := x.LatLong(); err == nil {
		result.Lat = &lat
		result.Lng = &lng
	}
	return result
}

func computeContentHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type imageDimensions struct {
	Width       int
	Height      int
	AspectRatio float64
	Orientation string
}

func extractDimensions(data []byte) (imageDimensions, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return imageDimensions{}, err
	}
	d := imageDimensions{Width: cfg.Width, Height: cfg.Height}
	if cfg.Height > 0 {
		d.AspectRatio = float64(cfg.Width) / float64(cfg.Height)
	}
	if cfg.Width >= cfg.Height {
		d.Orientation = "landscape"
	} else {
		d.Orientation = "portrait"
	}
	return d, nil
}

func computePHash(data []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	h, err := goimagehash.PerceptionHash(img)
	if err != nil {
		return "", err
	}
	return h.ToString(), nil
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
