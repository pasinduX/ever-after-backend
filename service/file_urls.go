package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	neturl "net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/storyvows/backend/integrations"
)

// fileURLs builds and re-signs S3 object URLs. Presigned GET URLs carry a TTL,
// so anything that hands a URL to a client — or to an external fetcher like the
// OpenAI vision API — must mint a fresh one at the moment of use rather than
// reusing the URL stored at upload time.
type fileURLs struct {
	cfg       *integrations.Secrets
	s3        *s3.Client
	presigner *s3.PresignClient
}

func newFileURLs(cfg *integrations.Secrets, s3Client *s3.Client) *fileURLs {
	var presigner *s3.PresignClient
	if s3Client != nil {
		presigner = s3.NewPresignClient(s3Client)
	}
	return &fileURLs{cfg: cfg, s3: s3Client, presigner: presigner}
}

// build returns a URL for the object, presigned when the bucket is private.
func (f *fileURLs) build(fileKey string) (string, error) {
	if f.cfg.S3Bucket == "" {
		return "", errors.New("s3 bucket is not configured")
	}

	if f.cfg.S3PublicBaseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(f.cfg.S3PublicBaseURL, "/"), fileKey), nil
	}

	rawURL := f.raw(fileKey)
	if f.s3 == nil || f.presigner == nil {
		return rawURL, nil
	}

	req, err := f.presigner.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(f.cfg.S3Bucket),
		Key:    aws.String(fileKey),
	}, func(po *s3.PresignOptions) {
		po.Expires = f.cfg.S3PresignTTL
	})
	if err != nil {
		slog.Warn("failed to create presigned url, falling back to raw public url", "error", err.Error(), "file_key", fileKey)
		return rawURL, nil
	}
	return req.URL, nil
}

// fresh is build() with errors flattened to an empty string, for call sites that
// treat a missing URL as "skip this one".
func (f *fileURLs) fresh(fileKey string) string {
	if fileKey == "" {
		return ""
	}
	url, err := f.build(fileKey)
	if err != nil {
		return ""
	}
	return url
}

// refresh re-signs a previously issued URL by recovering its object key. URLs
// that aren't ours to re-sign (public base URL, external links) come back
// unchanged.
func (f *fileURLs) refresh(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	fileKey, ok := f.extractKey(rawURL)
	if !ok {
		return rawURL
	}
	fresh, err := f.build(fileKey)
	if err != nil || fresh == "" {
		return rawURL
	}
	return fresh
}

// extractKey recovers the S3 object key from a URL previously produced by
// raw()/PresignGetObject, so it can be re-signed at serve time.
func (f *fileURLs) extractKey(rawURL string) (string, bool) {
	if f.cfg.S3Bucket == "" {
		return "", false
	}
	u, err := neturl.Parse(rawURL)
	if err != nil {
		return "", false
	}

	if strings.HasPrefix(u.Host, f.cfg.S3Bucket+".s3.") && strings.HasSuffix(u.Host, ".amazonaws.com") {
		return strings.TrimPrefix(u.Path, "/"), true
	}

	if f.cfg.S3Endpoint != "" {
		if endpointHost, err := neturl.Parse(f.cfg.S3Endpoint); err == nil && endpointHost.Host == u.Host {
			prefix := "/" + f.cfg.S3Bucket + "/"
			if strings.HasPrefix(u.Path, prefix) {
				return strings.TrimPrefix(u.Path, prefix), true
			}
		}
	}

	return "", false
}

func (f *fileURLs) raw(fileKey string) string {
	if f.cfg.S3PublicBaseURL != "" {
		return fmt.Sprintf("%s/%s", strings.TrimRight(f.cfg.S3PublicBaseURL, "/"), fileKey)
	}

	if f.cfg.S3Endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimRight(f.cfg.S3Endpoint, "/"), f.cfg.S3Bucket, fileKey)
	}

	if f.cfg.S3Region == "us-east-1" {
		return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", f.cfg.S3Bucket, fileKey)
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", f.cfg.S3Bucket, f.cfg.S3Region, fileKey)
}
