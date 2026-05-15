#!/usr/bin/env python3
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent

files = {
    'integrations/secrets.go': '''package integrations

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

    S3Endpoint        string
    S3Bucket          string
    S3Region          string
    S3AccessKeyID     string
    S3SecretAccessKey string
    S3PublicBaseURL   string

    FrontendURL   string
    MaxUploadSize int64
}

func Load() (*Secrets, error) {
    _ = godotenv.Load()
    s := &Secrets{
        Port:                 getEnv("PORT", "8080"),
        Env:                  getEnv("ENV", "development"),
        MongoURL:             requireEnv("MONGO_DB_URL"),
        MongoDBName:          getEnv("MONGO_DB_NAME", "everafter"),
        JWTSecret:            requireEnv("JWT_SECRET"),
        JWTAccessTokenTTL:    parseDuration(getEnv("JWT_ACCESS_TTL", "15m")),
        JWTRefreshTokenTTL:   parseDuration(getEnv("JWT_REFRESH_TTL", "720h")),
        StripeSecretKey:      requireEnv("STRIPE_SECRET_KEY"),
        StripeWebhookSecret:  requireEnv("STRIPE_WEBHOOK_SECRET"),
        StripeElopementPrice: parseInt64(getEnv("STRIPE_ELOPEMENT_PRICE", "19900")),
        StripeHeritagePrice:  parseInt64(getEnv("STRIPE_HERITAGE_PRICE", "44900")),
        StripeLegacyPrice:    parseInt64(getEnv("STRIPE_LEGACY_PRICE", "79900")),
        S3Endpoint:           getEnv("S3_ENDPOINT", ""),
        S3Bucket:             requireEnv("S3_BUCKET"),
        S3Region:             getEnv("S3_REGION", "auto"),
        S3AccessKeyID:        requireEnv("S3_ACCESS_KEY_ID"),
        S3SecretAccessKey:    requireEnv("S3_SECRET_ACCESS_KEY"),
        S3PublicBaseURL:      requireEnv("S3_PUBLIC_BASE_URL"),
        FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:3000"),
        MaxUploadSize:        parseInt64(getEnv("MAX_UPLOAD_SIZE", "52428800")),
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
''',

    'dbConfig/dbConnector.go': '''package dbConfig

import (
    "context"
    "fmt"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    "go.mongodb.org/mongo-driver/mongo/readpref"
)

func Connect(ctx context.Context, mongoURL, dbName string) (*mongo.Client, *mongo.Database, error) {
    client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURL))
    if err != nil {
        return nil, nil, fmt.Errorf("connect mongo: %w", err)
    }
    if err := client.Ping(ctx, readpref.Primary()); err != nil {
        return nil, nil, fmt.Errorf("ping mongo: %w", err)
    }
    return client, client.Database(dbName), nil
}
''',

    'dto/model_User.go': '''package dto

import "time"

type User struct {
    ID           string    `json:"id" bson:"_id,omitempty"`
    Email        string    `json:"email" bson:"email"`
    PasswordHash string    `json:"-" bson:"password_hash"`
    FullName     string    `json:"full_name" bson:"full_name"`
    CreatedAt    time.Time `json:"created_at" bson:"created_at"`
    UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
}

type RefreshToken struct {
    ID        string    `json:"id" bson:"_id,omitempty"`
    UserID    string    `json:"user_id" bson:"user_id"`
    TokenHash string    `json:"-" bson:"token_hash"`
    ExpiresAt time.Time `json:"expires_at" bson:"expires_at"`
    CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type SignUpRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
    FullName string `json:"full_name"`
}

type SignInRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type RefreshRequest struct {
    RefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    User         *User  `json:"user"`
}
''',

    'dto/model_Wedding.go': '''package dto

import "time"

type Tier string

const (
    TierElopement Tier = "elopement"
    TierHeritage  Tier = "heritage"
    TierLegacy    Tier = "legacy"
)

type Privacy string

const (
    PrivacyPublic            Privacy = "public"
    PrivacyPrivate           Privacy = "private"
    PrivacyPasswordProtected Privacy = "password_protected"
)

type Wedding struct {
    ID             string     `json:"id" bson:"_id,omitempty"`
    OwnerID        string     `json:"owner_id" bson:"owner_id"`
    CoupleNames    string     `json:"couple_names" bson:"couple_names"`
    WeddingDate    time.Time  `json:"wedding_date" bson:"wedding_date"`
    Venue          string     `json:"venue" bson:"venue"`
    WelcomeMessage string     `json:"welcome_message" bson:"welcome_message"`
    QRSlug         string     `json:"qr_slug" bson:"qr_slug"`
    QRCodeURL      string     `json:"qr_code_url" bson:"qr_code_url"`
    Tier           Tier       `json:"tier" bson:"tier"`
    Privacy        Privacy    `json:"privacy" bson:"privacy"`
    PasswordHash   *string    `json:"-" bson:"password_hash,omitempty"`
    IsActive       bool       `json:"is_active" bson:"is_active"`
    ExpiresAt      *time.Time `json:"expires_at,omitempty" bson:"expires_at,omitempty"`
    CreatedAt      time.Time  `json:"created_at" bson:"created_at"`
    UpdatedAt      time.Time  `json:"updated_at" bson:"updated_at"`
    UploadCount    int        `json:"upload_count,omitempty" bson:"upload_count,omitempty"`
}

func (w *Wedding) UploadLimit() int {
    if w.Tier == TierElopement {
        return 100
    }
    return -1
}

type CreateWeddingRequest struct {
    CoupleNames    string `json:"couple_names"`
    WeddingDate    string `json:"wedding_date"`
    Venue          string `json:"venue"`
    WelcomeMessage string `json:"welcome_message"`
}

type UpdateWeddingRequest struct {
    CoupleNames    *string `json:"couple_names"`
    WeddingDate    *string `json:"wedding_date"`
    Venue          *string `json:"venue"`
    WelcomeMessage *string `json:"welcome_message"`
}

type PrivacyRequest struct {
    Privacy  Privacy `json:"privacy"`
    Password *string `json:"password"`
}

type GuestAccessRequest struct {
    Password string `json:"password"`
}
''',

    'dto/model_Upload.go': '''package dto

import "time"

type UploadCategory string

const (
    CategoryCeremony UploadCategory = "ceremony"
    CategoryCandid   UploadCategory = "candid"
    CategoryDancing  UploadCategory = "dancing"
    CategoryFamily   UploadCategory = "family"
    CategoryOther    UploadCategory = "other"
)

type FileType string

const (
    FileTypePhoto FileType = "photo"
    FileTypeVideo FileType = "video"
)

type Upload struct {
    ID         string         `json:"id" bson:"_id,omitempty"`
    WeddingID  string         `json:"wedding_id" bson:"wedding_id"`
    GuestName  *string        `json:"guest_name,omitempty" bson:"guest_name,omitempty"`
    FileURL    string         `json:"file_url" bson:"file_url"`
    FileKey    string         `json:"-" bson:"file_key"`
    FileType   FileType       `json:"file_type" bson:"file_type"`
    MimeType   string         `json:"mime_type" bson:"mime_type"`
    SizeBytes  int64          `json:"size_bytes" bson:"size_bytes"`
    Category   UploadCategory `json:"category" bson:"category"`
    IsApproved bool           `json:"is_approved" bson:"is_approved"`
    UploadedAt time.Time      `json:"uploaded_at" bson:"uploaded_at"`
}

type ApproveUploadRequest struct {
    Approved bool `json:"approved"`
}
''',

    'dto/model_Order.go': '''package dto

import "time"

type OrderStatus string

const (
    OrderStatusPending  OrderStatus = "pending"
    OrderStatusPaid     OrderStatus = "paid"
    OrderStatusRefunded OrderStatus = "refunded"
)

type Order struct {
    ID                    string      `json:"id" bson:"_id,omitempty"`
    WeddingID             string      `json:"wedding_id" bson:"wedding_id"`
    UserID                string      `json:"user_id" bson:"user_id"`
    Tier                  Tier        `json:"tier" bson:"tier"`
    AmountCents           int64       `json:"amount_cents" bson:"amount_cents"`
    Currency              string      `json:"currency" bson:"currency"`
    Status                OrderStatus `json:"status" bson:"status"`
    StripeSessionID       string      `json:"stripe_session_id" bson:"stripe_session_id"`
    StripePaymentIntentID *string     `json:"stripe_payment_intent_id,omitempty" bson:"stripe_payment_intent_id,omitempty"`
    PaidAt                *time.Time  `json:"paid_at,omitempty" bson:"paid_at,omitempty"`
    ExpiresAt             *time.Time  `json:"expires_at,omitempty" bson:"expires_at,omitempty"`
    CreatedAt             time.Time   `json:"created_at" bson:"created_at"`
}

type CheckoutRequest struct {
    WeddingID string `json:"wedding_id"`
    Tier      Tier   `json:"tier"`
}

type CheckoutResponse struct {
    CheckoutURL string `json:"checkout_url"`
    SessionID   string `json:"session_id"`
}

type ErrorResponse struct {
    Error   string `json:"error"`
    Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"`
}
''',
}

# Helper write function

def write(path, content):
    full = ROOT / path
    full.parent.mkdir(parents=True, exist_ok=True)
    full.write_text(content, encoding='utf-8')
    print('Wrote', path)

for path, content in files.items():
    write(path, content)

# DAO file contents

dao_files = {
    'dao/dao_create_user.go': '''package dao

import (
    "context"
    "fmt"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/mongo"
)

func CreateUser(ctx context.Context, db *mongo.Database, user *dto.User) error {
    _, err := db.Collection("users").InsertOne(ctx, user)
    if err != nil {
        return fmt.Errorf("dao CreateUser: %w", err)
    }
    return nil
}
''',

    'dao/dao_find_user.go': '''package dao

import (
    "context"
    "fmt"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
)

func FindUserByEmail(ctx context.Context, db *mongo.Database, email string) (*dto.User, error) {
    var user dto.User
    err := db.Collection("users").FindOne(ctx, bson.M{"email": email}).Decode(&user)
    if err != nil {
        return nil, fmt.Errorf("dao FindUserByEmail: %w", err)
    }
    return &user, nil
}

func FindUserByID(ctx context.Context, db *mongo.Database, id string) (*dto.User, error) {
    var user dto.User
    err := db.Collection("users").FindOne(ctx, bson.M{"_id": id}).Decode(&user)
    if err != nil {
        return nil, fmt.Errorf("dao FindUserByID: %w", err)
    }
    return &user, nil
}

func CountUsersByEmail(ctx context.Context, db *mongo.Database, email string) (int, error) {
    count, err := db.Collection("users").CountDocuments(ctx, bson.M{"email": email})
    if err != nil {
        return 0, fmt.Errorf("dao CountUsersByEmail: %w", err)
    }
    return int(count), nil
}

var ErrNoRows = mongo.ErrNoDocuments
''',

    'dao/dao_create_refreshtoken.go': '''package dao

import (
    "context"
    "fmt"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/mongo"
)

func CreateRefreshToken(ctx context.Context, db *mongo.Database, rt *dto.RefreshToken) error {
    _, err := db.Collection("refresh_tokens").InsertOne(ctx, rt)
    if err != nil {
        return fmt.Errorf("dao CreateRefreshToken: %w", err)
    }
    return nil
}
''',

    'dao/dao_find_refreshtoken.go': '''package dao

import (
    "context"
    "fmt"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
)

func FindRefreshTokenByHash(ctx context.Context, db *mongo.Database, tokenHash string) (*dto.RefreshToken, error) {
    var rt dto.RefreshToken
    err := db.Collection("refresh_tokens").FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(&rt)
    if err != nil {
        return nil, fmt.Errorf("dao FindRefreshTokenByHash: %w", err)
    }
    return &rt, nil
}
''',

    'dao/dao_delete_refreshtoken.go': '''package dao

import (
    "context"
    "fmt"

    "go.mongodb.org/mongo-driver/bson"
)

func DeleteRefreshTokenByID(ctx context.Context, db *mongo.Database, id string) error {
    _, err := db.Collection("refresh_tokens").DeleteOne(ctx, bson.M{"_id": id})
    if err != nil {
        return fmt.Errorf("dao DeleteRefreshTokenByID: %w", err)
    }
    return nil
}

func DeleteRefreshTokenByHash(ctx context.Context, db *mongo.Database, tokenHash string) error {
    _, err := db.Collection("refresh_tokens").DeleteOne(ctx, bson.M{"token_hash": tokenHash})
    if err != nil {
        return fmt.Errorf("dao DeleteRefreshTokenByHash: %w", err)
    }
    return nil
}
''',

    'dao/dao_create_wedding.go': '''package dao

import (
    "context"
    "fmt"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/mongo"
)

func CreateWedding(ctx context.Context, db *mongo.Database, w *dto.Wedding) error {
    _, err := db.Collection("weddings").InsertOne(ctx, w)
    if err != nil {
        return fmt.Errorf("dao CreateWedding: %w", err)
    }
    return nil
}
''',

    'dao/dao_find_wedding.go': '''package dao

import (
    "context"
    "fmt"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/bson"
)

func FindWeddingByID(ctx context.Context, db *mongo.Database, id string) (*dto.Wedding, error) {
    var w dto.Wedding
    err := db.Collection("weddings").FindOne(ctx, bson.M{"_id": id}).Decode(&w)
    if err != nil {
        return nil, fmt.Errorf("dao FindWeddingByID: %w", err)
    }
    return &w, nil
}

func FindWeddingBySlug(ctx context.Context, db *mongo.Database, slug string) (*dto.Wedding, error) {
    var w dto.Wedding
    err := db.Collection("weddings").FindOne(ctx, bson.M{"qr_slug": slug}).Decode(&w)
    if err != nil {
        return nil, fmt.Errorf("dao FindWeddingBySlug: %w", err)
    }
    return &w, nil
}
''',

    'dao/dao_findall_wedding.go': '''package dao

import (
    "context"
    "fmt"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/bson"
)

func FindWeddingsByOwner(ctx context.Context, db *mongo.Database, ownerID string) ([]*dto.Wedding, error) {
    cursor, err := db.Collection("weddings").Find(ctx, bson.M{"owner_id": ownerID})
    if err != nil {
        return nil, fmt.Errorf("dao FindWeddingsByOwner: %w", err)
    }
    defer cursor.Close(ctx)

    var weddings []*dto.Wedding
    if err := cursor.All(ctx, &weddings); err != nil {
        return nil, fmt.Errorf("dao FindWeddingsByOwner decode: %w", err)
    }
    return weddings, nil
}
''',

    'dao/dao_update_wedding.go': '''package dao

import (
    "context"
    "fmt"
    "time"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/bson"
)

func UpdateWedding(ctx context.Context, db *mongo.Database, w *dto.Wedding) error {
    w.UpdatedAt = time.Now()
    _, err := db.Collection("weddings").ReplaceOne(ctx, bson.M{"_id": w.ID}, w)
    if err != nil {
        return fmt.Errorf("dao UpdateWedding: %w", err)
    }
    return nil
}

func UpdateWeddingPrivacy(ctx context.Context, db *mongo.Database, weddingID string, privacy dto.Privacy, passwordHash *string) error {
    update := bson.M{"privacy": privacy}
    if passwordHash != nil {
        update["password_hash"] = *passwordHash
    } else {
        update["password_hash"] = nil
    }
    _, err := db.Collection("weddings").UpdateOne(ctx, bson.M{"_id": weddingID}, bson.M{"$set": update})
    if err != nil {
        return fmt.Errorf("dao UpdateWeddingPrivacy: %w", err)
    }
    return nil
}

func ActivateWeddingTier(ctx context.Context, db *mongo.Database, weddingID string, tier dto.Tier, expiresAt *time.Time) error {
    update := bson.M{"tier": tier, "expires_at": expiresAt}
    _, err := db.Collection("weddings").UpdateOne(ctx, bson.M{"_id": weddingID}, bson.M{"$set": update})
    if err != nil {
        return fmt.Errorf("dao ActivateWeddingTier: %w", err)
    }
    return nil
}
''',

    'dao/dao_delete_wedding.go': '''package dao

import (
    "context"
    "fmt"

    "go.mongodb.org/mongo-driver/bson"
)

func DeactivateWedding(ctx context.Context, db *mongo.Database, weddingID string) error {
    _, err := db.Collection("weddings").UpdateOne(ctx, bson.M{"_id": weddingID}, bson.M{"$set": bson.M{"is_active": false}})
    if err != nil {
        return fmt.Errorf("dao DeactivateWedding: %w", err)
    }
    return nil
}
''',

    'dao/dao_create_upload.go': '''package dao

import (
    "context"
    "fmt"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/mongo"
)

func CreateUpload(ctx context.Context, db *mongo.Database, u *dto.Upload) error {
    _, err := db.Collection("uploads").InsertOne(ctx, u)
    if err != nil {
        return fmt.Errorf("dao CreateUpload: %w", err)
    }
    return nil
}
''',

    'dao/dao_find_upload.go': '''package dao

import (
    "context"
    "fmt"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/bson"
)

func FindUploadByID(ctx context.Context, db *mongo.Database, id string) (*dto.Upload, error) {
    var u dto.Upload
    err := db.Collection("uploads").FindOne(ctx, bson.M{"_id": id}).Decode(&u)
    if err != nil {
        return nil, fmt.Errorf("dao FindUploadByID: %w", err)
    }
    return &u, nil
}

func CountUploadsByWedding(ctx context.Context, db *mongo.Database, weddingID string) (int, error) {
    count, err := db.Collection("uploads").CountDocuments(ctx, bson.M{"wedding_id": weddingID})
    if err != nil {
        return 0, fmt.Errorf("dao CountUploadsByWedding: %w", err)
    }
    return int(count), nil
}
''',

    'dao/dao_findall_upload.go': '''package dao

import (
    "context"
    "fmt"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
)

func FindUploadsByWedding(ctx context.Context, db *mongo.Database, weddingID string) ([]*dto.Upload, error) {
    cursor, err := db.Collection("uploads").Find(ctx, bson.M{"wedding_id": weddingID})
    if err != nil {
        return nil, fmt.Errorf("dao FindUploadsByWedding: %w", err)
    }
    defer cursor.Close(ctx)

    var uploads []*dto.Upload
    if err := cursor.All(ctx, &uploads); err != nil {
        return nil, fmt.Errorf("dao FindUploadsByWedding decode: %w", err)
    }
    return uploads, nil
}

func FindApprovedUploadsByWedding(ctx context.Context, db *mongo.Database, weddingID string) ([]*dto.Upload, error) {
    cursor, err := db.Collection("uploads").Find(ctx, bson.M{"wedding_id": weddingID, "is_approved": true})
    if err != nil {
        return nil, fmt.Errorf("dao FindApprovedUploadsByWedding: %w", err)
    }
    defer cursor.Close(ctx)

    var uploads []*dto.Upload
    if err := cursor.All(ctx, &uploads); err != nil {
        return nil, fmt.Errorf("dao FindApprovedUploadsByWedding decode: %w", err)
    }
    return uploads, nil
}

func FindRandomPhotoHighlights(ctx context.Context, db *mongo.Database, weddingID string, limit int) ([]*dto.Upload, error) {
    pipeline := mongo.Pipeline{
        { {"$match", bson.M{"wedding_id": weddingID, "is_approved": true, "file_type": "photo"}} },
        { {"$sample", bson.M{"size": limit}} },
    }
    cursor, err := db.Collection("uploads").Aggregate(ctx, pipeline)
    if err != nil {
        return nil, fmt.Errorf("dao FindRandomPhotoHighlights: %w", err)
    }
    defer cursor.Close(ctx)

    var uploads []*dto.Upload
    if err := cursor.All(ctx, &uploads); err != nil {
        return nil, fmt.Errorf("dao FindRandomPhotoHighlights decode: %w", err)
    }
    return uploads, nil
}
''',

    'dao/dao_update_upload.go': '''package dao

import (
    "context"
    "fmt"

    "go.mongodb.org/mongo-driver/bson"
)

func SetUploadApproval(ctx context.Context, db *mongo.Database, uploadID string, approved bool) error {
    _, err := db.Collection("uploads").UpdateOne(ctx, bson.M{"_id": uploadID}, bson.M{"$set": bson.M{"is_approved": approved}})
    if err != nil {
        return fmt.Errorf("dao SetUploadApproval: %w", err)
    }
    return nil
}
''',

    'dao/dao_delete_upload.go': '''package dao

import (
    "context"
    "fmt"

    "go.mongodb.org/mongo-driver/bson"
)

func DeleteUpload(ctx context.Context, db *mongo.Database, uploadID string) error {
    _, err := db.Collection("uploads").DeleteOne(ctx, bson.M{"_id": uploadID})
    if err != nil {
        return fmt.Errorf("dao DeleteUpload: %w", err)
    }
    return nil
}
''',

    'dao/dao_create_order.go': '''package dao

import (
    "context"
    "fmt"

    "github.com/storyvows/backend/dto"
    "go.mongodb.org/mongo-driver/mongo"
)

func CreateOrder(ctx context.Context, db *mongo.Database, o *dto.Order) error {
    _, err := db.Collection("orders").InsertOne(ctx, o)
    if err != nil {
        return fmt.Errorf("dao CreateOrder: %w", err)
    }
    return nil
}
''',

    'dao/dao_update_order.go': '''package dao

import (
    "context"
    "fmt"
    "time"

    "go.mongodb.org/mongo-driver/bson"
)

func MarkOrderPaid(ctx context.Context, db *mongo.Database, stripeSessionID, paymentIntentID string) error {
    update := bson.M{"status": "paid", "paid_at": time.Now()}
    if paymentIntentID != "" {
        update["stripe_payment_intent_id"] = paymentIntentID
    }
    _, err := db.Collection("orders").UpdateOne(ctx, bson.M{"stripe_session_id": stripeSessionID}, bson.M{"$set": update})
    if err != nil {
        return fmt.Errorf("dao MarkOrderPaid: %w", err)
    }
    return nil
}
''',
}

for path, content in dao_files.items():
    write(path, content)

# Service files
service_files = {
    'service/auth_service.go': '''package service

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "github.com/storyvows/backend/dao"
    "github.com/storyvows/backend/dto"
    apperrors "github.com/storyvows/backend/errors"
    "github.com/storyvows/backend/functions"
    "github.com/storyvows/backend/integrations"
    "go.mongodb.org/mongo-driver/mongo"
)

type AuthService struct {
    db  *mongo.Database
    cfg *integrations.Secrets
}

func NewAuthService(db *mongo.Database, cfg *integrations.Secrets) *AuthService {
    return &AuthService{db: db, cfg: cfg}
}

func (s *AuthService) SignUp(ctx context.Context, req dto.SignUpRequest) (*dto.AuthResponse, error) {
    count, err := dao.CountUsersByEmail(ctx, s.db, req.Email)
    if err != nil {
        return nil, fmt.Errorf("auth SignUp count: %w", err)
    }
    if count > 0 {
        return nil, apperrors.ErrEmailTaken
    }

    hash, err := functions.HashPassword(req.Password)
    if err != nil {
        return nil, fmt.Errorf("auth SignUp hash: %w", err)
    }

    now := time.Now()
    user := &dto.User{
        ID:           uuid.NewString(),
        Email:        req.Email,
        PasswordHash: hash,
        FullName:     req.FullName,
        CreatedAt:    now,
        UpdatedAt:    now,
    }
    if err := dao.CreateUser(ctx, s.db, user); err != nil {
        return nil, fmt.Errorf("auth SignUp create: %w", err)
    }
    return s.issueTokens(ctx, user)
}

func (s *AuthService) SignIn(ctx context.Context, req dto.SignInRequest) (*dto.AuthResponse, error) {
    user, err := dao.FindUserByEmail(ctx, s.db, req.Email)
    if err != nil {
        if errors.Is(err, dao.ErrNoRows) {
            return nil, apperrors.ErrInvalidCreds
        }
        return nil, fmt.Errorf("auth SignIn find: %w", err)
    }
    if !functions.CheckPassword(req.Password, user.PasswordHash) {
        return nil, apperrors.ErrInvalidCreds
    }
    return s.issueTokens(ctx, user)
}

func (s *AuthService) RefreshTokens(ctx context.Context, rawToken string) (*dto.AuthResponse, error) {
    hash := functions.HashToken(rawToken)
    rt, err := dao.FindRefreshTokenByHash(ctx, s.db, hash)
    if err != nil {
        if errors.Is(err, dao.ErrNoRows) {
            return nil, apperrors.ErrInvalidToken
        }
        return nil, fmt.Errorf("auth Refresh find: %w", err)
    }
    if time.Now().After(rt.ExpiresAt) {
        _ = dao.DeleteRefreshTokenByID(ctx, s.db, rt.ID)
        return nil, apperrors.ErrInvalidToken
    }
    _ = dao.DeleteRefreshTokenByID(ctx, s.db, rt.ID)
    user, err := dao.FindUserByID(ctx, s.db, rt.UserID)
    if err != nil {
        return nil, fmt.Errorf("auth Refresh find user: %w", err)
    }
    return s.issueTokens(ctx, user)
}

func (s *AuthService) SignOut(ctx context.Context, rawToken string) error {
    hash := functions.HashToken(rawToken)
    if err := dao.DeleteRefreshTokenByHash(ctx, s.db, hash); err != nil {
        return fmt.Errorf("auth SignOut delete: %w", err)
    }
    return nil
}

func (s *AuthService) Me(ctx context.Context, userID string) (*dto.User, error) {
    user, err := dao.FindUserByID(ctx, s.db, userID)
    if err != nil {
        return nil, fmt.Errorf("auth Me find: %w", err)
    }
    return user, nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *dto.User) (*dto.AuthResponse, error) {
    now := time.Now()
    accessClaims := jwt.MapClaims{
        "sub": user.ID,
        "exp": now.Add(s.cfg.JWTAccessTokenTTL).Unix(),
        "iat": now.Unix(),
    }
    accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTSecret))
    if err != nil {
        return nil, fmt.Errorf("sign access token: %w", err)
    }

    rawRefresh := uuid.NewString()
    rt := &dto.RefreshToken{
        ID:        uuid.NewString(),
        UserID:    user.ID,
        TokenHash: functions.HashToken(rawRefresh),
        ExpiresAt: now.Add(s.cfg.JWTRefreshTokenTTL),
        CreatedAt: now,
    }
    if err := dao.CreateRefreshToken(ctx, s.db, rt); err != nil {
        return nil, fmt.Errorf("store refresh token: %w", err)
    }

    return &dto.AuthResponse{
        AccessToken:  accessToken,
        RefreshToken: rawRefresh,
        User:         user,
    }, nil
}
''',

    'service/wedding_service.go': '''package service

import (
    "context"
    "errors"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/storyvows/backend/dao"
    "github.com/storyvows/backend/dto"
    apperrors "github.com/storyvows/backend/errors"
    "github.com/storyvows/backend/functions"
    "github.com/storyvows/backend/integrations"
    "go.mongodb.org/mongo-driver/mongo"
)

type WeddingService struct {
    db  *mongo.Database
    cfg *integrations.Secrets
}

func NewWeddingService(db *mongo.Database, cfg *integrations.Secrets) *WeddingService {
    return &WeddingService{db: db, cfg: cfg}
}

func (s *WeddingService) Create(ctx context.Context, ownerID string, req dto.CreateWeddingRequest) (*dto.Wedding, error) {
    if req.CoupleNames == "" {
        return nil, errors.New("couple_names is required")
    }
    weddingDate, err := time.Parse("2006-01-02", req.WeddingDate)
    if err != nil {
        return nil, errors.New("wedding_date must be in YYYY-MM-DD format")
    }

    slug := functions.GenerateSlug()
    qrURL := fmt.Sprintf("%s/w/%s", s.cfg.FrontendURL, slug)
    qrCodeURL, _ := functions.GenerateQRDataURL(qrURL)

    now := time.Now()
    w := &dto.Wedding{
        ID:             uuid.NewString(),
        OwnerID:        ownerID,
        CoupleNames:    req.CoupleNames,
        WeddingDate:    weddingDate,
        Venue:          req.Venue,
        WelcomeMessage: req.WelcomeMessage,
        QRSlug:         slug,
        QRCodeURL:      qrCodeURL,
        Tier:           dto.TierElopement,
        Privacy:        dto.PrivacyPublic,
        IsActive:       true,
        CreatedAt:      now,
        UpdatedAt:      now,
    }
    if err := dao.CreateWedding(ctx, s.db, w); err != nil {
        return nil, err
    }
    return w, nil
}

func (s *WeddingService) List(ctx context.Context, ownerID string) ([]*dto.Wedding, error) {
    return dao.FindWeddingsByOwner(ctx, s.db, ownerID)
}

func (s *WeddingService) Get(ctx context.Context, weddingID, ownerID string) (*dto.Wedding, error) {
    w, err := dao.FindWeddingByID(ctx, s.db, weddingID)
    if err != nil {
        return nil, err
    }
    if w.OwnerID != ownerID {
        return nil, apperrors.ErrForbidden
    }
    return w, nil
}

func (s *WeddingService) GetBySlug(ctx context.Context, slug string) (*dto.Wedding, error) {
    return dao.FindWeddingBySlug(ctx, s.db, slug)
}

func (s *WeddingService) Update(ctx context.Context, weddingID, ownerID string, req dto.UpdateWeddingRequest) (*dto.Wedding, error) {
    w, err := s.Get(ctx, weddingID, ownerID)
    if err != nil {
        return nil, err
    }
    if req.CoupleNames != nil {
        w.CoupleNames = *req.CoupleNames
    }
    if req.Venue != nil {
        w.Venue = *req.Venue
    }
    if req.WelcomeMessage != nil {
        w.WelcomeMessage = *req.WelcomeMessage
    }
    if req.WeddingDate != nil {
        d, err := time.Parse("2006-01-02", *req.WeddingDate)
        if err != nil {
            return nil, errors.New("wedding_date must be YYYY-MM-DD")
        }
        w.WeddingDate = d
    }
    if err := dao.UpdateWedding(ctx, s.db, w); err != nil {
        return nil, err
    }
    return w, nil
}

func (s *WeddingService) Delete(ctx context.Context, weddingID, ownerID string) error {
    if _, err := s.Get(ctx, weddingID, ownerID); err != nil {
        return err
    }
    return dao.DeactivateWedding(ctx, s.db, weddingID)
}

func (s *WeddingService) SetPrivacy(ctx context.Context, weddingID, ownerID string, req dto.PrivacyRequest) error {
    if _, err := s.Get(ctx, weddingID, ownerID); err != nil {
        return err
    }
    var passwordHash *string
    if req.Privacy == dto.PrivacyPasswordProtected {
        if req.Password == nil || *req.Password == "" {
            return errors.New("password required for password_protected privacy")
        }
        h, err := functions.HashPassword(*req.Password)
        if err != nil {
            return err
        }
        passwordHash = &h
    }
    return dao.UpdateWeddingPrivacy(ctx, s.db, weddingID, req.Privacy, passwordHash)
}

func (s *WeddingService) VerifyGuestAccess(ctx context.Context, slug, password string) (*dto.Wedding, error) {
    w, err := dao.FindWeddingBySlug(ctx, s.db, slug)
    if err != nil {
        return nil, err
    }
    if w.Privacy == dto.PrivacyPrivate {
        return nil, apperrors.ErrForbidden
    }
    if w.Privacy == dto.PrivacyPasswordProtected {
        if w.PasswordHash == nil || !functions.CheckPassword(*w.PasswordHash, password) {
            return nil, errors.New("incorrect album password")
        }
    }
    return w, nil
}

func (s *WeddingService) ActivateTier(ctx context.Context, weddingID string, tier dto.Tier, expiresAt *time.Time) error {
    return dao.ActivateWeddingTier(ctx, s.db, weddingID, tier, expiresAt)
}
''',

    'service/upload_service.go': '''package service

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
    "github.com/storyvows/backend/functions"
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
    _, _ = s3.DeleteObject(ctx, &s3.DeleteObjectInput{
        Bucket: aws.String(s.cfg.S3Bucket),
        Key:    aws.String(upload.FileKey),
    })
    return dao.DeleteUpload(ctx, s.db, uploadID)
}
''',

    'service/payment_service.go': '''package service

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/storyvows/backend/dao"
    "github.com/storyvows/backend/dto"
    apperrors "github.com/storyvows/backend/errors"
    "github.com/storyvows/backend/integrations"
    "github.com/stripe/stripe-go/v82"
    "github.com/stripe/stripe-go/v82/checkout/session"
    "go.mongodb.org/mongo-driver/mongo"
)

type PaymentService struct {
    db  *mongo.Database
    cfg *integrations.Secrets
}

func NewPaymentService(db *mongo.Database, cfg *integrations.Secrets) *PaymentService {
    stripe.Key = cfg.StripeSecretKey
    return &PaymentService{db: db, cfg: cfg}
}

func (s *PaymentService) CreateCheckout(ctx context.Context, userID string, req dto.CheckoutRequest) (*dto.CheckoutResponse, error) {
    amount, name := s.tierDetails(req.Tier)
    if amount == 0 {
        return nil, apperrors.ErrInvalidTier
    }

    params := &stripe.CheckoutSessionParams{
        Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
        LineItems: []*stripe.CheckoutSessionLineItemParams{
            {
                PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
                    Currency:   stripe.String("usd"),
                    UnitAmount: stripe.Int64(amount),
                    ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
                        Name: stripe.String(fmt.Sprintf("Story Vows — %s", name)),
                    },
                },
                Quantity: stripe.Int64(1),
            },
        },
        SuccessURL: stripe.String(fmt.Sprintf("%s/dashboard?payment=success", s.cfg.FrontendURL)),
        CancelURL:  stripe.String(fmt.Sprintf("%s/pricing?payment=cancelled", s.cfg.FrontendURL)),
        Metadata: map[string]string{
            "user_id":    userID,
            "wedding_id": req.WeddingID,
            "tier":       string(req.Tier),
        },
    }

    sess, err := session.New(params)
    if err != nil {
        return nil, fmt.Errorf("create stripe session: %w", err)
    }

    order := &dto.Order{
        ID:              uuid.NewString(),
        WeddingID:       req.WeddingID,
        UserID:          userID,
        Tier:            req.Tier,
        AmountCents:     amount,
        Currency:        "usd",
        StripeSessionID: sess.ID,
        Status:          dto.OrderStatusPending,
        CreatedAt:       time.Now(),
    }
    if err := dao.CreateOrder(ctx, s.db, order); err != nil {
        return nil, fmt.Errorf("save order: %w", err)
    }

    return &dto.CheckoutResponse{
        CheckoutURL: sess.URL,
        SessionID:   sess.ID,
    }, nil
}

func (s *PaymentService) HandleWebhook(ctx context.Context, event stripe.Event) error {
    if event.Type != "checkout.session.completed" {
        return nil
    }

    sess := &stripe.CheckoutSession{}
    if err := json.Unmarshal(event.Data.Raw, sess); err != nil {
        return nil, fmt.Errorf("parse session: %w", err)
    }

    weddingID := sess.Metadata["wedding_id"]
    tier := dto.Tier(sess.Metadata["tier"])
    paymentIntentID := ""
    if sess.PaymentIntent != nil {
        paymentIntentID = sess.PaymentIntent.ID
    }

    if err := dao.MarkOrderPaid(ctx, s.db, sess.ID, paymentIntentID); err != nil {
        return nil, fmt.Errorf("update order: %w", err)
    }

    var expiresAt *time.Time
    if tier == dto.TierElopement {
        t := time.Now().AddDate(1, 0, 0)
        expiresAt = &t
    }

    if err := dao.ActivateWeddingTier(ctx, s.db, weddingID, tier, expiresAt); err != nil {
        return nil, fmt.Errorf("activate tier: %w", err)
    }
    return nil
}

func (s *PaymentService) tierDetails(tier dto.Tier) (int64, string) {
    switch tier {
    case dto.TierElopement:
        return s.cfg.StripeElopementPrice, "Elopement"
    case dto.TierHeritage:
        return s.cfg.StripeHeritagePrice, "Heritage"
    case dto.TierLegacy:
        return s.cfg.StripeLegacyPrice, "Legacy"
    }
    return 0, ""
}
''',
}

for path, content in service_files.items():
    write(path, content)

misc = {
    'api/api_gallery.go': '''package api

import (
    "archive/zip"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/storyvows/backend/dao"
    "github.com/storyvows/backend/dto"
    "github.com/storyvows/backend/utils"
    "go.mongodb.org/mongo-driver/mongo"
)

func Album(db *mongo.Database) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        weddingID := chi.URLParam(r, "id")
        uploads, err := dao.FindApprovedUploadsByWedding(r.Context(), db, weddingID)
        if err != nil {
            utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to load album")
            return
        }
        album := map[string][]*dto.Upload{}
        for _, u := range uploads {
            album[string(u.Category)] = append(album[string(u.Category)], u)
        }
        utils.SendJSON(w, http.StatusOK, album)
    }
}

func Highlights(db *mongo.Database) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        weddingID := chi.URLParam(r, "id")
        highlights, err := dao.FindRandomPhotoHighlights(r.Context(), db, weddingID, 20)
        if err != nil {
            utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to load highlights")
            return
        }
        utils.SendJSON(w, http.StatusOK, highlights)
    }
}

func Download(db *mongo.Database) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        weddingID := chi.URLParam(r, "id")

        wedding, err := dao.FindWeddingByID(r.Context(), db, weddingID)
        if err != nil {
            utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to load wedding")
            return
        }
        if wedding.Tier == dto.TierElopement {
            utils.SendErrorResponse(w, http.StatusPaymentRequired, "bulk download requires Heritage or Legacy tier")
            return
        }

        uploads, err := dao.FindApprovedUploadsByWedding(r.Context(), db, weddingID)
        if err != nil {
            utils.SendErrorResponse(w, http.StatusInternalServerError, "failed to load uploads")
            return
        }

        w.Header().Set("Content-Type", "application/zip")
        w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-album.zip\"", weddingID))
        zw := zip.NewWriter(w)
        defer zw.Close()

        client := &http.Client{Timeout: 30 * time.Second}
        for _, upload := range uploads {
            resp, err := client.Get(upload.FileURL)
            if err != nil {
                continue
            }
            parsed, _ := url.Parse(upload.FileURL)
            fileName := fmt.Sprintf("%s%s", upload.ID, parsedExtension(parsed.Path))
            f, err := zw.Create(fileName)
            if err != nil {
                resp.Body.Close()
                continue
            }
            _, _ = io.Copy(f, resp.Body)
            resp.Body.Close()
        }
    }
}

func parsedExtension(path string) string {
    for i := len(path) - 1; i >= 0; i-- {
        if path[i] == '.' {
            return path[i:]
        }
        if path[i] == '/' {
            break
        }
    }
    return ""
}
''',

    'apiHandlers/router.go': '''package apiHandlers

import (
    "fmt"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    chiMiddleware "github.com/go-chi/chi/v5/middleware"
    "github.com/go-chi/cors"
    "github.com/go-chi/httprate"
    "go.mongodb.org/mongo-driver/mongo"
    "github.com/storyvows/backend/api"
    "github.com/storyvows/backend/functions"
    "github.com/storyvows/backend/integrations"
    "github.com/storyvows/backend/realtime"
    "github.com/storyvows/backend/service"
)

func NewRouter(
    cfg *integrations.Secrets,
    db *mongo.Database,
    authSvc *service.AuthService,
    weddingSvc *service.WeddingService,
    uploadSvc *service.UploadService,
    paymentSvc *service.PaymentService,
    hub *realtime.Hub,
) http.Handler {
    r := chi.NewRouter()

    r.Use(chiMiddleware.Recoverer)
    r.Use(functions.Logger)
    r.Use(cors.Handler(cors.Options{
        AllowedOrigins:   []string{cfg.FrontendURL},
        AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
        AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
        AllowCredentials: true,
        MaxAge:           300,
    }))
    r.Use(httprate.LimitAll(200, time.Minute))

    r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = fmt.Fprintf(w, `{"status":"ok"}`)
    })

    requireAuth := functions.RequireAuth(cfg.JWTSecret)
    getUID := functions.GetUserID

    r.Route("/api/auth", func(r chi.Router) {
        r.Post("/signup", api.SignUp(authSvc))
        r.Post("/signin", api.SignIn(authSvc))
        r.Post("/refresh", api.Refresh(authSvc))
        r.With(requireAuth).Post("/signout", api.SignOut(authSvc))
        r.With(requireAuth).Get("/me", api.Me(authSvc, getUID))
    })

    r.Route("/api/weddings", func(r chi.Router) {
        r.Use(requireAuth)
        r.Post("/", api.CreateWedding(weddingSvc, getUID))
        r.Get("/", api.ListWeddings(weddingSvc, getUID))
        r.Get("/{id}", api.GetWedding(weddingSvc, getUID))
        r.Patch("/{id}", api.UpdateWedding(weddingSvc, getUID))
        r.Delete("/{id}", api.DeleteWedding(weddingSvc, getUID))
        r.Patch("/{id}/privacy", api.SetPrivacyWedding(weddingSvc, getUID))
        r.Get("/{id}/uploads", api.ListUploads(uploadSvc))
        r.Patch("/{id}/uploads/{uploadId}/approve", api.ApproveUpload(uploadSvc))
        r.Delete("/{id}/uploads/{uploadId}", api.DeleteUpload(uploadSvc))
        r.Get("/{id}/album", api.Album(db))
        r.Get("/{id}/album/highlights", api.Highlights(db))
        r.Get("/{id}/album/download", api.Download(db))
        r.Get("/{id}/wall", hub.ServeSSE)
    })

    r.Route("/api/w/{slug}", func(r chi.Router) {
        r.Get("/", api.GuestViewWedding(weddingSvc))
        r.Post("/access", api.GuestAccessWedding(weddingSvc))
        r.Post("/uploads", api.GuestUpload(uploadSvc, hub, cfg.MaxUploadSize))
    })

    r.With(requireAuth).Post("/api/checkout", api.Checkout(paymentSvc, getUID))
    r.Post("/api/webhooks/stripe", api.StripeWebhook(paymentSvc, cfg))

    return r
}
''',

    'main.go': '''package main

import (
    "context"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/storyvows/backend/apiHandlers"
    "github.com/storyvows/backend/dbConfig"
    "github.com/storyvows/backend/integrations"
    "github.com/storyvows/backend/realtime"
    "github.com/storyvows/backend/service"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    slog.SetDefault(logger)

    cfg, err := integrations.Load()
    if err != nil {
        slog.Error("failed to load config", "error", err)
        os.Exit(1)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, db, err := dbConfig.Connect(ctx, cfg.MongoURL, cfg.MongoDBName)
    if err != nil {
        slog.Error("failed to connect to database", "error", err)
        os.Exit(1)
    }
    defer func() {
        _ = client.Disconnect(context.Background())
    }()

    authSvc := service.NewAuthService(db, cfg)
    weddingSvc := service.NewWeddingService(db, cfg)
    uploadSvc, err := service.NewUploadService(db, cfg)
    if err != nil {
        slog.Error("failed to init upload service", "error", err)
        os.Exit(1)
    }
    paymentSvc := service.NewPaymentService(db, cfg)
    hub := realtime.NewHub()

    router := apiHandlers.NewRouter(cfg, db, authSvc, weddingSvc, uploadSvc, paymentSvc, hub)

    srv := &http.Server{
        Addr:         ":" + cfg.Port,
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    slog.Info("starting server", "port", cfg.Port, "env", cfg.Env)
    go func() {
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            slog.Error("server error", "error", err)
            os.Exit(1)
        }
    }()

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    slog.Info("shutting down...")

    shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer shutCancel()
    _ = srv.Shutdown(shutCtx)
    slog.Info("shutdown complete")
}
''',
}

for path, content in misc.items():
    write(path, content)

all_paths = list(files.keys()) + list(dao_files.keys()) + list(service_files.keys()) + list(misc.keys())
subprocess.run(["gofmt", "-w"] + all_paths, check=True)
print("Rewrote files and ran gofmt.")
