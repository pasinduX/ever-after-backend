package integrations

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Secrets struct {
	Port string
	Env  string

	MongoURL    string
	MongoDBName string

	JWTSecret          string
	JWTAccessTokenTTL  time.Duration
	JWTRefreshTokenTTL time.Duration

	StripeSecretKey      string
	StripeWebhookSecret  string
	StripeElopementPrice int64
	StripeHeritagePrice  int64
	StripeLegacyPrice    int64

	SendGridAPIKey        string
	SendGridFromEmail     string
	SendGridDataResidency string

	S3Endpoint        string
	S3Bucket          string
	S3Region          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3PublicBaseURL   string

	FrontendURL         string
	FrontendCORSOrigins string
	MaxUploadSize       int64
}

func Load() (*Secrets, error) {
	_ = godotenv.Load()
	s := &Secrets{
		Port:                  getEnv("PORT", "8080"),
		Env:                   getEnv("ENV", "development"),
		MongoURL:              requireEnv("MONGO_DB_URL"),
		MongoDBName:           getEnv("MONGO_DB_NAME", "everafter"),
		JWTSecret:             requireEnv("JWT_SECRET"),
		JWTAccessTokenTTL:     parseDuration(getEnv("JWT_ACCESS_TTL", "15m")),
		JWTRefreshTokenTTL:    parseDuration(getEnv("JWT_REFRESH_TTL", "720h")),
		StripeSecretKey:       requireEnv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:   requireEnv("STRIPE_WEBHOOK_SECRET"),
		StripeElopementPrice:  parseInt64(getEnv("STRIPE_ELOPEMENT_PRICE", "19900")),
		StripeHeritagePrice:   parseInt64(getEnv("STRIPE_HERITAGE_PRICE", "44900")),
		StripeLegacyPrice:     parseInt64(getEnv("STRIPE_LEGACY_PRICE", "79900")),
		SendGridAPIKey:        requireEnv("SENDGRID_API_KEY"),
		SendGridFromEmail:     requireEnv("SENDGRID_FROM_EMAIL"),
		SendGridDataResidency: getEnv("SENDGRID_DATA_RESIDENCY", ""),
		S3Endpoint:            getEnv("S3_ENDPOINT", ""),
		S3Bucket:              requireEnv("S3_BUCKET"),
		S3Region:              getEnv("S3_REGION", "auto"),
		S3AccessKeyID:         requireEnv("S3_ACCESS_KEY_ID"),
		S3SecretAccessKey:     requireEnv("S3_SECRET_ACCESS_KEY"),
		S3PublicBaseURL:       requireEnv("S3_PUBLIC_BASE_URL"),
		FrontendURL:           getEnv("FRONTEND_URL", "http://localhost:3000"),
		FrontendCORSOrigins:   getEnv("FRONTEND_ORIGINS", getEnv("FRONTEND_URL", "http://localhost:3000")),
		MaxUploadSize:         parseInt64(getEnv("MAX_UPLOAD_SIZE", "52428800")),
	}
	return s, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(fmt.Sprintf("invalid duration %q: %v", s, err))
	}
	return d
}

func parseInt64(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
