package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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
	db       *mongo.Database
	cfg      *integrations.Secrets
	s3       *s3.Client
	urls     *fileURLs
	analysis *AnalysisService
}

func NewUploadService(db *mongo.Database, cfg *integrations.Secrets, s3Client *s3.Client, analysis *AnalysisService) (*UploadService, error) {
	return &UploadService{db: db, cfg: cfg, s3: s3Client, urls: newFileURLs(cfg, s3Client), analysis: analysis}, nil
}

// FreshURL generates a new presigned GET URL for the given S3 file key.
// Call this at serve time so URLs are never stale when delivered to the client.
func (s *UploadService) FreshURL(fileKey string) string { return s.urls.fresh(fileKey) }

// RefreshURL re-signs a previously issued file URL using the S3 object key
// embedded in it, so callers never serve back an expired presigned GET URL.
// URLs that aren't ours to re-sign are returned unchanged.
func (s *UploadService) RefreshURL(rawURL string) string { return s.urls.refresh(rawURL) }

func (s *UploadService) buildFileURL(fileKey string) (string, error) { return s.urls.build(fileKey) }

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
		count, err := dao.CountUploadsByWedding(ctx, s.db, weddingID)
		if err != nil {
			return nil, fmt.Errorf("count uploads: %w", err)
		}
		if count >= limit {
			return nil, apperrors.ErrLimitReached
		}
	}

	return s.store(ctx, weddingID, fmt.Sprintf("weddings/%s", weddingID), file, header, mimeType, fileType, guestName)
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

	return s.store(ctx, folderID, strings.Trim(folderID, "/"), file, header, mimeType, fileType, "")
}

// store is the single persistence path shared by every upload entry point: read
// the bytes once, derive everything from one decode, push the original plus its
// derivatives to S3, then record the row and queue analysis.
func (s *UploadService) store(
	ctx context.Context,
	weddingID string,
	keyPrefix string,
	file multipart.File,
	header *multipart.FileHeader,
	mimeType string,
	fileType dto.FileType,
	guestName string,
) (*dto.Upload, error) {
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	exifData := extractEXIF(fileBytes)
	contentHash := computeContentHash(fileBytes)

	// One decode covers dimensions, perceptual hash and both derivatives.
	var media mediaAnalysis
	if fileType == dto.FileTypePhoto {
		media = analyseImage(fileBytes)
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	baseName := uuid.NewString()
	fileKey := fmt.Sprintf("%s/%s%s", keyPrefix, baseName, ext)

	if err := s.putObject(ctx, fileKey, fileBytes, mimeType); err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	fileURL, err := s.buildFileURL(fileKey)
	if err != nil {
		return nil, err
	}

	// Derivatives are an optimisation, never a hard requirement — if one fails
	// to store we serve the original for that size rather than failing the whole
	// upload the guest is waiting on.
	thumbnailURL := fileURL
	mediumURL := fileURL
	if key, ok := s.putDerivative(ctx, keyPrefix, baseName, "thumb", media.Thumbnail); ok {
		if url, err := s.buildFileURL(key); err == nil {
			thumbnailURL = url
		}
	}
	if key, ok := s.putDerivative(ctx, keyPrefix, baseName, "medium", media.Medium); ok {
		if url, err := s.buildFileURL(key); err == nil {
			mediumURL = url
		}
	}

	now := time.Now()
	timeline := dto.UploadTimeline{UploadedAt: now}
	if exifData.CapturedAt != nil {
		timeline.CapturedAt = exifData.CapturedAt
	}

	thumbStage := dto.AnalysisStatusSucceeded
	if !media.Decoded {
		thumbStage = dto.AnalysisStatusPending
	}

	dims := media.Dims
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
			MediumURL:    mediumURL,
			ThumbnailURL: thumbnailURL,
			FileKey:      fileKey,
		},
		Metadata: dto.UploadMetadata{
			MimeType:    mimeType,
			SizeBytes:   header.Size,
			Width:       intPtrOrNil(dims.Width),
			Height:      intPtrOrNil(dims.Height),
			AspectRatio: floatPtrOrNil(dims.AspectRatio),
			Orientation: stringPtrOrNil(dims.Orientation),
		},
		Timeline:    timeline,
		EXIF:        exifData,
		PHash:       media.PHash,
		Orientation: stringPtrOrNil(dims.Orientation),
		Analysis: dto.UploadAnalysis{
			Status:   dto.AnalysisStatusPending,
			Category: dto.CategoryOther,
			Processing: dto.ProcessingStages{
				Thumbnail:      thumbStage,
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

func (s *UploadService) putObject(ctx context.Context, key string, body []byte, contentType string) error {
	_, err := s.s3.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.cfg.S3Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(contentType),
	})
	return err
}

// putDerivative stores a downscaled variant. It reports false when there was
// nothing to store (small source, or an image we couldn't decode) or the put
// failed, in which case the caller keeps pointing at the original.
func (s *UploadService) putDerivative(ctx context.Context, keyPrefix, baseName, variant string, body []byte) (string, bool) {
	if len(body) == 0 {
		return "", false
	}
	key := fmt.Sprintf("%s/%s_%s.jpg", keyPrefix, baseName, variant)
	if err := s.putObject(ctx, key, body, "image/jpeg"); err != nil {
		slog.Warn("failed to store image derivative, falling back to original",
			"error", err.Error(), "variant", variant, "key", key)
		return "", false
	}
	return key, true
}

func intPtrOrNil(v int) *int {
	if v <= 0 {
		return nil
	}
	return &v
}

func floatPtrOrNil(v float64) *float64 {
	if v <= 0 {
		return nil
	}
	return &v
}

func stringPtrOrNil(v string) *string {
	if v == "" {
		return nil
	}
	return &v
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

func (s *UploadService) ListForWedding(ctx context.Context, weddingID string) ([]*dto.Upload, error) {
	return dao.FindUploadsByWedding(ctx, s.db, weddingID)
}

func (s *UploadService) SetApproval(ctx context.Context, uploadID string, approved bool) error {
	return dao.SetUploadApproval(ctx, s.db, uploadID, approved)
}

// DeleteObjectsForWedding removes every stored object belonging to a wedding,
// originals and derivatives alike.
//
// Best-effort by design: a leftover object costs pennies, whereas returning an
// error here would block the owner from ever completing the delete. It must run
// before the upload rows are removed, since the rows are the only record of
// which keys exist.
func (s *UploadService) DeleteObjectsForWedding(ctx context.Context, weddingID string) {
	if s.s3 == nil || s.cfg.S3Bucket == "" {
		return
	}

	uploads, err := dao.FindUploadsByWedding(ctx, s.db, weddingID)
	if err != nil {
		slog.Warn("could not list uploads for object cleanup", "error", err.Error(), "wedding_id", weddingID)
		return
	}

	seen := make(map[string]struct{}, len(uploads)*3)
	var keys []types.ObjectIdentifier
	add := func(key string) {
		if key == "" {
			return
		}
		if _, dup := seen[key]; dup {
			return
		}
		seen[key] = struct{}{}
		keys = append(keys, types.ObjectIdentifier{Key: aws.String(key)})
	}

	for _, u := range uploads {
		add(u.FileKey)
		add(u.Storage.FileKey)
		// Derivative keys aren't stored directly, but they're recoverable from
		// the URLs we issued for them.
		if key, ok := s.urls.extractKey(u.Storage.ThumbnailURL); ok {
			add(key)
		}
		if key, ok := s.urls.extractKey(u.Storage.MediumURL); ok {
			add(key)
		}
	}

	// S3 caps DeleteObjects at 1000 keys per request.
	const batchSize = 1000
	for start := 0; start < len(keys); start += batchSize {
		end := start + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		_, err := s.s3.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: aws.String(s.cfg.S3Bucket),
			Delete: &types.Delete{Objects: keys[start:end], Quiet: aws.Bool(true)},
		})
		if err != nil {
			slog.Warn("failed to delete stored objects for wedding",
				"error", err.Error(), "wedding_id", weddingID, "batch_start", start)
		}
	}
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
