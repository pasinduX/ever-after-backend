#!/usr/bin/env python3
"""Script to write Go source files cleanly."""
import os

BASE = "/Users/pasindurathnayaka/Documents/wedding/story-vows-backend"

files = {}

# ── models/models.go ──────────────────────────────────────────────────────────
files["internal/models/models.go"] = """\
package models

import "time"

type Tier string

const (
\tTierElopement Tier = "elopement"
\tTierHeritage  Tier = "heritage"
\tTierLegacy    Tier = "legacy"
)

type Privacy string

const (
\tPrivacyPublic            Privacy = "public"
\tPrivacyPrivate           Privacy = "private"
\tPrivacyPasswordProtected Privacy = "password_protected"
)

type UploadCategory string

const (
\tCategoryCeremony UploadCategory = "ceremony"
\tCategoryCandid   UploadCategory = "candid"
\tCategoryDancing  UploadCategory = "dancing"
\tCategoryFamily   UploadCategory = "family"
\tCategoryOther    UploadCategory = "other"
)

type FileType string

const (
\tFileTypePhoto FileType = "photo"
\tFileTypeVideo FileType = "video"
)

type OrderStatus string

const (
\tOrderStatusPending  OrderStatus = "pending"
\tOrderStatusPaid     OrderStatus = "paid"
\tOrderStatusRefunded OrderStatus = "refunded"
)

// User is a registered couple account.
type User struct {
\tID           string    `json:"id" db:"id"`
\tEmail        string    `json:"email" db:"email"`
\tPasswordHash string    `json:"-" db:"password_hash"`
\tFullName     string    `json:"full_name" db:"full_name"`
\tCreatedAt    time.Time `json:"created_at" db:"created_at"`
\tUpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// RefreshToken stores hashed refresh tokens for a user.
type RefreshToken struct {
\tID        string    `json:"id" db:"id"`
\tUserID    string    `json:"user_id" db:"user_id"`
\tTokenHash string    `json:"-" db:"token_hash"`
\tExpiresAt time.Time `json:"expires_at" db:"expires_at"`
\tCreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Wedding is a wedding event owned by a couple.
type Wedding struct {
\tID             string     `json:"id" db:"id"`
\tOwnerID        string     `json:"owner_id" db:"owner_id"`
\tCoupleNames    string     `json:"couple_names" db:"couple_names"`
\tWeddingDate    time.Time  `json:"wedding_date" db:"wedding_date"`
\tVenue          string     `json:"venue" db:"venue"`
\tWelcomeMessage string     `json:"welcome_message" db:"welcome_message"`
\tQRSlug         string     `json:"qr_slug" db:"qr_slug"`
\tQRCodeURL      string     `json:"qr_code_url" db:"qr_code_url"`
\tTier           Tier       `json:"tier" db:"tier"`
\tPrivacy        Privacy    `json:"privacy" db:"privacy"`
\tPasswordHash   *string    `json:"-" db:"password_hash"`
\tIsActive       bool       `json:"is_active" db:"is_active"`
\tExpiresAt      *time.Time `json:"expires_at,omitempty" db:"expires_at"`
\tCreatedAt      time.Time  `json:"created_at" db:"created_at"`
\tUpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
\tUploadCount    int        `json:"upload_count,omitempty" db:"upload_count"`
}

// UploadLimit returns the max number of uploads for this wedding tier.
// Returns -1 for unlimited.
func (w *Wedding) UploadLimit() int {
\tif w.Tier == TierElopement {
\t\treturn 100
\t}
\treturn -1
}

// Upload is a guest-submitted photo or video.
type Upload struct {
\tID         string         `json:"id" db:"id"`
\tWeddingID  string         `json:"wedding_id" db:"wedding_id"`
\tGuestName  *string        `json:"guest_name,omitempty" db:"guest_name"`
\tFileURL    string         `json:"file_url" db:"file_url"`
\tFileKey    string         `json:"-" db:"file_key"`
\tFileType   FileType       `json:"file_type" db:"file_type"`
\tMimeType   string         `json:"mime_type" db:"mime_type"`
\tSizeBytes  int64          `json:"size_bytes" db:"size_bytes"`
\tCategory   UploadCategory `json:"category" db:"category"`
\tIsApproved bool           `json:"is_approved" db:"is_approved"`
\tUploadedAt time.Time      `json:"uploaded_at" db:"uploaded_at"`
}

// Order records a one-time purchase for a wedding tier.
type Order struct {
\tID                    string      `json:"id" db:"id"`
\tWeddingID             string      `json:"wedding_id" db:"wedding_id"`
\tUserID                string      `json:"user_id" db:"user_id"`
\tTier                  Tier        `json:"tier" db:"tier"`
\tAmountCents           int64       `json:"amount_cents" db:"amount_cents"`
\tCurrency              string      `json:"currency" db:"currency"`
\tStatus                OrderStatus `json:"status" db:"status"`
\tStripeSessionID       string      `json:"stripe_session_id" db:"stripe_session_id"`
\tStripePaymentIntentID *string     `json:"stripe_payment_intent_id,omitempty" db:"stripe_payment_intent_id"`
\tPaidAt                *time.Time  `json:"paid_at,omitempty" db:"paid_at"`
\tExpiresAt             *time.Time  `json:"expires_at,omitempty" db:"expires_at"`
\tCreatedAt             time.Time   `json:"created_at" db:"created_at"`
}

// ── DTOs ──────────────────────────────────────────────────────────────────────

type SignUpRequest struct {
\tEmail    string `json:"email"`
\tPassword string `json:"password"`
\tFullName string `json:"full_name"`
}

type SignInRequest struct {
\tEmail    string `json:"email"`
\tPassword string `json:"password"`
}

type AuthResponse struct {
\tAccessToken  string `json:"access_token"`
\tRefreshToken string `json:"refresh_token"`
\tUser         *User  `json:"user"`
}

type RefreshRequest struct {
\tRefreshToken string `json:"refresh_token"`
}

type CreateWeddingRequest struct {
\tCoupleNames    string `json:"couple_names"`
\tWeddingDate    string `json:"wedding_date"` // YYYY-MM-DD
\tVenue          string `json:"venue"`
\tWelcomeMessage string `json:"welcome_message"`
}

type UpdateWeddingRequest struct {
\tCoupleNames    *string `json:"couple_names"`
\tWeddingDate    *string `json:"wedding_date"`
\tVenue          *string `json:"venue"`
\tWelcomeMessage *string `json:"welcome_message"`
}

type PrivacyRequest struct {
\tPrivacy  Privacy `json:"privacy"`
\tPassword *string `json:"password"`
}

type GuestAccessRequest struct {
\tPassword string `json:"password"`
}

type CheckoutRequest struct {
\tWeddingID string `json:"wedding_id"`
\tTier      Tier   `json:"tier"`
}

type CheckoutResponse struct {
\tCheckoutURL string `json:"checkout_url"`
\tSessionID   string `json:"session_id"`
}

type ApproveUploadRequest struct {
\tApproved bool `json:"approved"`
}

type ErrorResponse struct {
\tError   string `json:"error"`
\tMessage string `json:"message,omitempty"`
}

type SuccessResponse struct {
\tMessage string `json:"message"`
\tData    any    `json:"data,omitempty"`
}
"""

# ── db/db.go ──────────────────────────────────────────────────────────────────
files["internal/db/db.go"] = """\
package db

import (
\t"context"
\t"fmt"

\t"github.com/jackc/pgx/v5/pgxpool"
)

// New creates a new pgx connection pool.
func New(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
\tconfig, err := pgxpool.ParseConfig(databaseURL)
\tif err != nil {
\t\treturn nil, fmt.Errorf("parse db config: %w", err)
\t}
\tconfig.MaxConns = 20
\tpool, err := pgxpool.NewWithConfig(ctx, config)
\tif err != nil {
\t\treturn nil, fmt.Errorf("create pool: %w", err)
\t}
\tif err := pool.Ping(ctx); err != nil {
\t\treturn nil, fmt.Errorf("ping db: %w", err)
\t}
\treturn pool, nil
}
"""

# ── middleware/middleware.go ───────────────────────────────────────────────────
files["internal/middleware/middleware.go"] = """\
package middleware

import (
\t"context"
\t"encoding/json"
\t"log/slog"
\t"net/http"
\t"strings"
\t"time"

\t"github.com/golang-jwt/jwt/v5"
\t"github.com/storyvows/backend/internal/models"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// Logger logs every request with method, path, status and duration.
func Logger(next http.Handler) http.Handler {
\treturn http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
\t\tstart := time.Now()
\t\twrapped := &statusRecorder{ResponseWriter: w, status: 200}
\t\tnext.ServeHTTP(wrapped, r)
\t\tslog.Info("request",
\t\t\t"method", r.Method,
\t\t\t"path", r.URL.Path,
\t\t\t"status", wrapped.status,
\t\t\t"duration", time.Since(start),
\t\t)
\t})
}

type statusRecorder struct {
\thttp.ResponseWriter
\tstatus int
}

func (sr *statusRecorder) WriteHeader(code int) {
\tsr.status = code
\tsr.ResponseWriter.WriteHeader(code)
}

// RequireAuth validates the Bearer JWT and injects user_id into context.
func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
\treturn func(next http.Handler) http.Handler {
\t\treturn http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
\t\t\tauthHeader := r.Header.Get("Authorization")
\t\t\tif !strings.HasPrefix(authHeader, "Bearer ") {
\t\t\t\twriteError(w, http.StatusUnauthorized, "missing or invalid authorization header")
\t\t\t\treturn
\t\t\t}
\t\t\ttokenStr := strings.TrimPrefix(authHeader, "Bearer ")

\t\t\ttoken, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
\t\t\t\tif _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
\t\t\t\t\treturn nil, jwt.ErrSignatureInvalid
\t\t\t\t}
\t\t\t\treturn []byte(jwtSecret), nil
\t\t\t})
\t\t\tif err != nil || !token.Valid {
\t\t\t\twriteError(w, http.StatusUnauthorized, "invalid or expired token")
\t\t\t\treturn
\t\t\t}

\t\t\tclaims, ok := token.Claims.(jwt.MapClaims)
\t\t\tif !ok {
\t\t\t\twriteError(w, http.StatusUnauthorized, "invalid token claims")
\t\t\t\treturn
\t\t\t}
\t\t\tuserID, ok := claims["sub"].(string)
\t\t\tif !ok {
\t\t\t\twriteError(w, http.StatusUnauthorized, "invalid token subject")
\t\t\t\treturn
\t\t\t}

\t\t\tctx := context.WithValue(r.Context(), UserIDKey, userID)
\t\t\tnext.ServeHTTP(w, r.WithContext(ctx))
\t\t})
\t}
}

// GetUserID extracts the authenticated user ID from context.
func GetUserID(r *http.Request) string {
\tuserID, _ := r.Context().Value(UserIDKey).(string)
\treturn userID
}

func writeError(w http.ResponseWriter, status int, msg string) {
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(status)
\t_ = json.NewEncoder(w).Encode(models.ErrorResponse{Error: msg})
}
"""

# ── auth/service.go ───────────────────────────────────────────────────────────
files["internal/auth/service.go"] = """\
package auth

import (
\t"context"
\t"crypto/sha256"
\t"encoding/hex"
\t"errors"
\t"fmt"
\t"time"

\t"github.com/golang-jwt/jwt/v5"
\t"github.com/google/uuid"
\t"github.com/jackc/pgx/v5"
\t"github.com/jackc/pgx/v5/pgxpool"
\t"github.com/storyvows/backend/internal/config"
\t"github.com/storyvows/backend/internal/models"
\t"golang.org/x/crypto/bcrypt"
)

var (
\tErrEmailTaken    = errors.New("email already in use")
\tErrInvalidCreds  = errors.New("invalid email or password")
\tErrInvalidToken  = errors.New("invalid or expired refresh token")
)

type Service struct {
\tdb  *pgxpool.Pool
\tcfg *config.Config
}

func NewService(db *pgxpool.Pool, cfg *config.Config) *Service {
\treturn &Service{db: db, cfg: cfg}
}

// SignUp creates a new user account.
func (s *Service) SignUp(ctx context.Context, req models.SignUpRequest) (*models.AuthResponse, error) {
\tif req.Email == "" || req.Password == "" || req.FullName == "" {
\t\treturn nil, errors.New("email, password and full_name are required")
\t}
\tif len(req.Password) < 8 {
\t\treturn nil, errors.New("password must be at least 8 characters")
\t}

\t// Check for duplicate email
\tvar count int
\terr := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE email = $1", req.Email).Scan(&count)
\tif err != nil {
\t\treturn nil, fmt.Errorf("check email: %w", err)
\t}
\tif count > 0 {
\t\treturn nil, ErrEmailTaken
\t}

\thash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
\tif err != nil {
\t\treturn nil, fmt.Errorf("hash password: %w", err)
\t}

\tuser := &models.User{
\t\tID:           uuid.NewString(),
\t\tEmail:        req.Email,
\t\tFullName:     req.FullName,
\t\tPasswordHash: string(hash),
\t}
\t_, err = s.db.Exec(ctx,
\t\t`INSERT INTO users (id, email, password_hash, full_name) VALUES ($1, $2, $3, $4)`,
\t\tuser.ID, user.Email, user.PasswordHash, user.FullName,
\t)
\tif err != nil {
\t\treturn nil, fmt.Errorf("insert user: %w", err)
\t}

\treturn s.issueTokens(ctx, user)
}

// SignIn authenticates a user and returns tokens.
func (s *Service) SignIn(ctx context.Context, req models.SignInRequest) (*models.AuthResponse, error) {
\tvar user models.User
\terr := s.db.QueryRow(ctx,
\t\t`SELECT id, email, password_hash, full_name, created_at, updated_at FROM users WHERE email = $1`,
\t\treq.Email,
\t).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.CreatedAt, &user.UpdatedAt)
\tif errors.Is(err, pgx.ErrNoRows) {
\t\treturn nil, ErrInvalidCreds
\t}
\tif err != nil {
\t\treturn nil, fmt.Errorf("query user: %w", err)
\t}
\tif bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
\t\treturn nil, ErrInvalidCreds
\t}
\treturn s.issueTokens(ctx, &user)
}

// RefreshTokens validates a refresh token and issues new token pair.
func (s *Service) RefreshTokens(ctx context.Context, rawToken string) (*models.AuthResponse, error) {
\ttokenHash := hashToken(rawToken)
\tvar rt models.RefreshToken
\terr := s.db.QueryRow(ctx,
\t\t`SELECT id, user_id, expires_at FROM refresh_tokens WHERE token_hash = $1`,
\t\ttokenHash,
\t).Scan(&rt.ID, &rt.UserID, &rt.ExpiresAt)
\tif errors.Is(err, pgx.ErrNoRows) || time.Now().After(rt.ExpiresAt) {
\t\treturn nil, ErrInvalidToken
\t}
\tif err != nil {
\t\treturn nil, fmt.Errorf("query refresh token: %w", err)
\t}

\t// Rotate: delete old token
\t_, _ = s.db.Exec(ctx, "DELETE FROM refresh_tokens WHERE id = $1", rt.ID)

\tvar user models.User
\terr = s.db.QueryRow(ctx,
\t\t`SELECT id, email, full_name, created_at, updated_at FROM users WHERE id = $1`,
\t\trt.UserID,
\t).Scan(&user.ID, &user.Email, &user.FullName, &user.CreatedAt, &user.UpdatedAt)
\tif err != nil {
\t\treturn nil, fmt.Errorf("query user for refresh: %w", err)
\t}
\treturn s.issueTokens(ctx, &user)
}

// SignOut revokes a refresh token.
func (s *Service) SignOut(ctx context.Context, rawToken string) error {
\ttokenHash := hashToken(rawToken)
\t_, err := s.db.Exec(ctx, "DELETE FROM refresh_tokens WHERE token_hash = $1", tokenHash)
\treturn err
}

// Me returns the authenticated user's profile.
func (s *Service) Me(ctx context.Context, userID string) (*models.User, error) {
\tvar user models.User
\terr := s.db.QueryRow(ctx,
\t\t`SELECT id, email, full_name, created_at, updated_at FROM users WHERE id = $1`,
\t\tuserID,
\t).Scan(&user.ID, &user.Email, &user.FullName, &user.CreatedAt, &user.UpdatedAt)
\tif err != nil {
\t\treturn nil, err
\t}
\treturn &user, nil
}

// issueTokens creates a JWT access token and a stored refresh token.
func (s *Service) issueTokens(ctx context.Context, user *models.User) (*models.AuthResponse, error) {
\tnow := time.Now()
\taccessClaims := jwt.MapClaims{
\t\t"sub":  user.ID,
\t\t"email": user.Email,
\t\t"iat":  now.Unix(),
\t\t"exp":  now.Add(s.cfg.JWTAccessTokenTTL).Unix(),
\t}
\taccess, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTSecret))
\tif err != nil {
\t\treturn nil, fmt.Errorf("sign access token: %w", err)
\t}

\trawRefresh := uuid.NewString()
\ttokenHash := hashToken(rawRefresh)
\texpiresAt := now.Add(s.cfg.JWTRefreshTokenTTL)
\t_, err = s.db.Exec(ctx,
\t\t`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at) VALUES ($1, $2, $3, $4)`,
\t\tuuid.NewString(), user.ID, tokenHash, expiresAt,
\t)
\tif err != nil {
\t\treturn nil, fmt.Errorf("store refresh token: %w", err)
\t}

\treturn &models.AuthResponse{
\t\tAccessToken:  access,
\t\tRefreshToken: rawRefresh,
\t\tUser:         user,
\t}, nil
}

func hashToken(token string) string {
\th := sha256.Sum256([]byte(token))
\treturn hex.EncodeToString(h[:])
}
"""

# ── auth/handler.go ───────────────────────────────────────────────────────────
files["internal/auth/handler.go"] = """\
package auth

import (
\t"encoding/json"
\t"errors"
\t"net/http"

\t"github.com/storyvows/backend/internal/middleware"
\t"github.com/storyvows/backend/internal/models"
)

type Handler struct {
\tsvc *Service
}

func NewHandler(svc *Service) *Handler {
\treturn &Handler{svc: svc}
}

func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
\tvar req models.SignUpRequest
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid request body")
\t\treturn
\t}
\tresp, err := h.svc.SignUp(r.Context(), req)
\tif errors.Is(err, ErrEmailTaken) {
\t\twriteError(w, http.StatusConflict, err.Error())
\t\treturn
\t}
\tif err != nil {
\t\twriteError(w, http.StatusBadRequest, err.Error())
\t\treturn
\t}
\twriteJSON(w, http.StatusCreated, resp)
}

func (h *Handler) SignIn(w http.ResponseWriter, r *http.Request) {
\tvar req models.SignInRequest
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid request body")
\t\treturn
\t}
\tresp, err := h.svc.SignIn(r.Context(), req)
\tif errors.Is(err, ErrInvalidCreds) {
\t\twriteError(w, http.StatusUnauthorized, err.Error())
\t\treturn
\t}
\tif err != nil {
\t\twriteError(w, http.StatusInternalServerError, "sign in failed")
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
\tvar req models.RefreshRequest
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid request body")
\t\treturn
\t}
\tresp, err := h.svc.RefreshTokens(r.Context(), req.RefreshToken)
\tif errors.Is(err, ErrInvalidToken) {
\t\twriteError(w, http.StatusUnauthorized, err.Error())
\t\treturn
\t}
\tif err != nil {
\t\twriteError(w, http.StatusInternalServerError, "refresh failed")
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SignOut(w http.ResponseWriter, r *http.Request) {
\tvar req models.RefreshRequest
\t_ = json.NewDecoder(r.Body).Decode(&req)
\t_ = h.svc.SignOut(r.Context(), req.RefreshToken)
\twriteJSON(w, http.StatusOK, models.SuccessResponse{Message: "signed out"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
\tuserID := middleware.GetUserID(r)
\tuser, err := h.svc.Me(r.Context(), userID)
\tif err != nil {
\t\twriteError(w, http.StatusNotFound, "user not found")
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, user)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(status)
\t_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
\twriteJSON(w, status, models.ErrorResponse{Error: msg})
}
"""

# ── wedding/service.go ────────────────────────────────────────────────────────
files["internal/wedding/service.go"] = """\
package wedding

import (
\t"bytes"
\t"context"
\t"encoding/base64"
\t"errors"
\t"fmt"
\t"strings"
\t"time"

\t"github.com/google/uuid"
\t"github.com/jackc/pgx/v5"
\t"github.com/jackc/pgx/v5/pgxpool"
\t"github.com/skip2/go-qrcode"
\t"github.com/storyvows/backend/internal/config"
\t"github.com/storyvows/backend/internal/models"
\t"golang.org/x/crypto/bcrypt"
)

var (
\tErrNotFound  = errors.New("wedding not found")
\tErrForbidden = errors.New("access denied")
)

type Service struct {
\tdb  *pgxpool.Pool
\tcfg *config.Config
}

func NewService(db *pgxpool.Pool, cfg *config.Config) *Service {
\treturn &Service{db: db, cfg: cfg}
}

// Create creates a new wedding for the given owner.
func (s *Service) Create(ctx context.Context, ownerID string, req models.CreateWeddingRequest) (*models.Wedding, error) {
\tif req.CoupleNames == "" {
\t\treturn nil, errors.New("couple_names is required")
\t}
\tweddingDate, err := time.Parse("2006-01-02", req.WeddingDate)
\tif err != nil {
\t\treturn nil, errors.New("wedding_date must be in YYYY-MM-DD format")
\t}

\tslug := generateSlug()
\tqrURL := fmt.Sprintf("%s/w/%s", s.cfg.FrontendURL, slug)
\tqrCodeURL, _ := generateQRDataURL(qrURL)

\tw := &models.Wedding{
\t\tID:             uuid.NewString(),
\t\tOwnerID:        ownerID,
\t\tCoupleNames:    req.CoupleNames,
\t\tWeddingDate:    weddingDate,
\t\tVenue:          req.Venue,
\t\tWelcomeMessage: req.WelcomeMessage,
\t\tQRSlug:         slug,
\t\tQRCodeURL:      qrCodeURL,
\t\tTier:           models.TierElopement, // default until purchase
\t\tPrivacy:        models.PrivacyPublic,
\t\tIsActive:       true,
\t}
\t_, err = s.db.Exec(ctx, `
\t\tINSERT INTO weddings
\t\t\t(id, owner_id, couple_names, wedding_date, venue, welcome_message, qr_slug, qr_code_url, tier, privacy, is_active)
\t\tVALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
\t\tw.ID, w.OwnerID, w.CoupleNames, w.WeddingDate, w.Venue, w.WelcomeMessage,
\t\tw.QRSlug, w.QRCodeURL, w.Tier, w.Privacy, w.IsActive,
\t)
\tif err != nil {
\t\treturn nil, fmt.Errorf("insert wedding: %w", err)
\t}
\treturn w, nil
}

// List returns all weddings owned by the given user.
func (s *Service) List(ctx context.Context, ownerID string) ([]*models.Wedding, error) {
\trows, err := s.db.Query(ctx, `
\t\tSELECT w.id, w.owner_id, w.couple_names, w.wedding_date, w.venue, w.welcome_message,
\t\t       w.qr_slug, w.qr_code_url, w.tier, w.privacy, w.is_active, w.expires_at, w.created_at, w.updated_at,
\t\t       COUNT(u.id) AS upload_count
\t\tFROM weddings w
\t\tLEFT JOIN uploads u ON u.wedding_id = w.id
\t\tWHERE w.owner_id = $1
\t\tGROUP BY w.id
\t\tORDER BY w.created_at DESC`, ownerID)
\tif err != nil {
\t\treturn nil, fmt.Errorf("list weddings: %w", err)
\t}
\tdefer rows.Close()

\tvar list []*models.Wedding
\tfor rows.Next() {
\t\tw := &models.Wedding{}
\t\tif err := rows.Scan(
\t\t\t&w.ID, &w.OwnerID, &w.CoupleNames, &w.WeddingDate, &w.Venue, &w.WelcomeMessage,
\t\t\t&w.QRSlug, &w.QRCodeURL, &w.Tier, &w.Privacy, &w.IsActive, &w.ExpiresAt,
\t\t\t&w.CreatedAt, &w.UpdatedAt, &w.UploadCount,
\t\t); err != nil {
\t\t\treturn nil, err
\t\t}
\t\tlist = append(list, w)
\t}
\treturn list, nil
}

// Get returns a single wedding by ID, enforcing ownership.
func (s *Service) Get(ctx context.Context, weddingID, ownerID string) (*models.Wedding, error) {
\tw, err := s.getByID(ctx, weddingID)
\tif err != nil {
\t\treturn nil, err
\t}
\tif w.OwnerID != ownerID {
\t\treturn nil, ErrForbidden
\t}
\treturn w, nil
}

// GetBySlug returns the public wedding details for guests.
func (s *Service) GetBySlug(ctx context.Context, slug string) (*models.Wedding, error) {
\tvar w models.Wedding
\terr := s.db.QueryRow(ctx, `
\t\tSELECT id, owner_id, couple_names, wedding_date, venue, welcome_message,
\t\t       qr_slug, qr_code_url, tier, privacy, password_hash, is_active, expires_at, created_at, updated_at
\t\tFROM weddings WHERE qr_slug = $1 AND is_active = true`, slug,
\t).Scan(
\t\t&w.ID, &w.OwnerID, &w.CoupleNames, &w.WeddingDate, &w.Venue, &w.WelcomeMessage,
\t\t&w.QRSlug, &w.QRCodeURL, &w.Tier, &w.Privacy, &w.PasswordHash, &w.IsActive, &w.ExpiresAt,
\t\t&w.CreatedAt, &w.UpdatedAt,
\t)
\tif errors.Is(err, pgx.ErrNoRows) {
\t\treturn nil, ErrNotFound
\t}
\treturn &w, err
}

// Update updates mutable fields of a wedding.
func (s *Service) Update(ctx context.Context, weddingID, ownerID string, req models.UpdateWeddingRequest) (*models.Wedding, error) {
\tw, err := s.Get(ctx, weddingID, ownerID)
\tif err != nil {
\t\treturn nil, err
\t}
\tif req.CoupleNames != nil {
\t\tw.CoupleNames = *req.CoupleNames
\t}
\tif req.Venue != nil {
\t\tw.Venue = *req.Venue
\t}
\tif req.WelcomeMessage != nil {
\t\tw.WelcomeMessage = *req.WelcomeMessage
\t}
\tif req.WeddingDate != nil {
\t\td, err := time.Parse("2006-01-02", *req.WeddingDate)
\t\tif err != nil {
\t\t\treturn nil, errors.New("wedding_date must be YYYY-MM-DD")
\t\t}
\t\tw.WeddingDate = d
\t}
\t_, err = s.db.Exec(ctx, `
\t\tUPDATE weddings
\t\tSET couple_names=$1, wedding_date=$2, venue=$3, welcome_message=$4, updated_at=NOW()
\t\tWHERE id=$5`,
\t\tw.CoupleNames, w.WeddingDate, w.Venue, w.WelcomeMessage, w.ID,
\t)
\treturn w, err
}

// Delete soft-deletes a wedding by deactivating it.
func (s *Service) Delete(ctx context.Context, weddingID, ownerID string) error {
\tw, err := s.Get(ctx, weddingID, ownerID)
\tif err != nil {
\t\treturn err
\t}
\t_, err = s.db.Exec(ctx, "UPDATE weddings SET is_active=false, updated_at=NOW() WHERE id=$1", w.ID)
\treturn err
}

// SetPrivacy updates the privacy mode and optional password.
func (s *Service) SetPrivacy(ctx context.Context, weddingID, ownerID string, req models.PrivacyRequest) error {
\tw, err := s.Get(ctx, weddingID, ownerID)
\tif err != nil {
\t\treturn err
\t}
\tvar passwordHash *string
\tif req.Privacy == models.PrivacyPasswordProtected {
\t\tif req.Password == nil || *req.Password == "" {
\t\t\treturn errors.New("password required for password_protected privacy")
\t\t}
\t\th, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
\t\tif err != nil {
\t\t\treturn err
\t\t}
\t\ts := string(h)
\t\tpasswordHash = &s
\t}
\t_, err = s.db.Exec(ctx,
\t\t"UPDATE weddings SET privacy=$1, password_hash=$2, updated_at=NOW() WHERE id=$3",
\t\treq.Privacy, passwordHash, w.ID,
\t)
\treturn err
}

// VerifyGuestAccess checks if a guest can access a password-protected wedding.
func (s *Service) VerifyGuestAccess(ctx context.Context, slug, password string) (*models.Wedding, error) {
\tw, err := s.GetBySlug(ctx, slug)
\tif err != nil {
\t\treturn nil, err
\t}
\tif w.Privacy == models.PrivacyPrivate {
\t\treturn nil, ErrForbidden
\t}
\tif w.Privacy == models.PrivacyPasswordProtected {
\t\tif w.PasswordHash == nil || bcrypt.CompareHashAndPassword([]byte(*w.PasswordHash), []byte(password)) != nil {
\t\t\treturn nil, errors.New("incorrect album password")
\t\t}
\t}
\treturn w, nil
}

// ActivateTier sets the tier and optional expiry after successful payment.
func (s *Service) ActivateTier(ctx context.Context, weddingID string, tier models.Tier, expiresAt *time.Time) error {
\t_, err := s.db.Exec(ctx,
\t\t"UPDATE weddings SET tier=$1, expires_at=$2, updated_at=NOW() WHERE id=$3",
\t\ttier, expiresAt, weddingID,
\t)
\treturn err
}

func (s *Service) getByID(ctx context.Context, id string) (*models.Wedding, error) {
\tvar w models.Wedding
\terr := s.db.QueryRow(ctx, `
\t\tSELECT id, owner_id, couple_names, wedding_date, venue, welcome_message,
\t\t       qr_slug, qr_code_url, tier, privacy, is_active, expires_at, created_at, updated_at
\t\tFROM weddings WHERE id = $1`, id,
\t).Scan(
\t\t&w.ID, &w.OwnerID, &w.CoupleNames, &w.WeddingDate, &w.Venue, &w.WelcomeMessage,
\t\t&w.QRSlug, &w.QRCodeURL, &w.Tier, &w.Privacy, &w.IsActive, &w.ExpiresAt,
\t\t&w.CreatedAt, &w.UpdatedAt,
\t)
\tif errors.Is(err, pgx.ErrNoRows) {
\t\treturn nil, ErrNotFound
\t}
\treturn &w, err
}

func generateSlug() string {
\traw := uuid.NewString()
\treturn strings.ReplaceAll(raw[:8], "-", "")
}

func generateQRDataURL(content string) (string, error) {
\tvar buf bytes.Buffer
\tpng, err := qrcode.Encode(content, qrcode.Medium, 256)
\tif err != nil {
\t\treturn "", err
\t}
\tbuf.WriteString("data:image/png;base64,")
\tbuf.WriteString(base64.StdEncoding.EncodeToString(png))
\treturn buf.String(), nil
}
"""

# ── wedding/handler.go ────────────────────────────────────────────────────────
files["internal/wedding/handler.go"] = """\
package wedding

import (
\t"encoding/json"
\t"errors"
\t"net/http"

\t"github.com/go-chi/chi/v5"
\t"github.com/storyvows/backend/internal/middleware"
\t"github.com/storyvows/backend/internal/models"
)

type Handler struct {
\tsvc *Service
}

func NewHandler(svc *Service) *Handler {
\treturn &Handler{svc: svc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
\tvar req models.CreateWeddingRequest
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid request body")
\t\treturn
\t}
\tresult, err := h.svc.Create(r.Context(), middleware.GetUserID(r), req)
\tif err != nil {
\t\twriteError(w, http.StatusBadRequest, err.Error())
\t\treturn
\t}
\twriteJSON(w, http.StatusCreated, result)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
\tlist, err := h.svc.List(r.Context(), middleware.GetUserID(r))
\tif err != nil {
\t\twriteError(w, http.StatusInternalServerError, "failed to list weddings")
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, list)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
\tid := chi.URLParam(r, "id")
\tresult, err := h.svc.Get(r.Context(), id, middleware.GetUserID(r))
\tif err != nil {
\t\twriteServiceError(w, err)
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
\tvar req models.UpdateWeddingRequest
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid request body")
\t\treturn
\t}
\tresult, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r), req)
\tif err != nil {
\t\twriteServiceError(w, err)
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
\tif err := h.svc.Delete(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r)); err != nil {
\t\twriteServiceError(w, err)
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, models.SuccessResponse{Message: "wedding deleted"})
}

func (h *Handler) SetPrivacy(w http.ResponseWriter, r *http.Request) {
\tvar req models.PrivacyRequest
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid request body")
\t\treturn
\t}
\tif err := h.svc.SetPrivacy(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r), req); err != nil {
\t\twriteServiceError(w, err)
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, models.SuccessResponse{Message: "privacy updated"})
}

// GuestView serves the public wedding page for guests (no auth).
func (h *Handler) GuestView(w http.ResponseWriter, r *http.Request) {
\tslug := chi.URLParam(r, "slug")
\tresult, err := h.svc.GetBySlug(r.Context(), slug)
\tif err != nil {
\t\twriteServiceError(w, err)
\t\treturn
\t}
\t// Never expose password hash to guests
\tresult.PasswordHash = nil
\twriteJSON(w, http.StatusOK, result)
}

// GuestAccess verifies a guest password for protected albums.
func (h *Handler) GuestAccess(w http.ResponseWriter, r *http.Request) {
\tvar req models.GuestAccessRequest
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid request body")
\t\treturn
\t}
\tresult, err := h.svc.VerifyGuestAccess(r.Context(), chi.URLParam(r, "slug"), req.Password)
\tif err != nil {
\t\twriteError(w, http.StatusUnauthorized, err.Error())
\t\treturn
\t}
\tresult.PasswordHash = nil
\twriteJSON(w, http.StatusOK, result)
}

func writeServiceError(w http.ResponseWriter, err error) {
\tswitch {
\tcase errors.Is(err, ErrNotFound):
\t\twriteError(w, http.StatusNotFound, err.Error())
\tcase errors.Is(err, ErrForbidden):
\t\twriteError(w, http.StatusForbidden, err.Error())
\tdefault:
\t\twriteError(w, http.StatusInternalServerError, err.Error())
\t}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(status)
\t_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
\twriteJSON(w, status, models.ErrorResponse{Error: msg})
}
"""

# ── upload/service.go ─────────────────────────────────────────────────────────
files["internal/upload/service.go"] = """\
package upload

import (
\t"context"
\t"errors"
\t"fmt"
\t"io"
\t"mime/multipart"
\t"path/filepath"
\t"strings"
\t"time"

\t"github.com/aws/aws-sdk-go-v2/aws"
\t"github.com/aws/aws-sdk-go-v2/credentials"
\t"github.com/aws/aws-sdk-go-v2/service/s3"
\tawsconfig "github.com/aws/aws-sdk-go-v2/config"
\t"github.com/google/uuid"
\t"github.com/jackc/pgx/v5"
\t"github.com/jackc/pgx/v5/pgxpool"
\t"github.com/storyvows/backend/internal/config"
\t"github.com/storyvows/backend/internal/models"
)

var (
\tallowedMimeTypes = map[string]models.FileType{
\t\t"image/jpeg": models.FileTypePhoto,
\t\t"image/png":  models.FileTypePhoto,
\t\t"image/webp": models.FileTypePhoto,
\t\t"image/heic": models.FileTypePhoto,
\t\t"video/mp4":  models.FileTypeVideo,
\t\t"video/mov":  models.FileTypeVideo,
\t\t"video/quicktime": models.FileTypeVideo,
\t}
\tErrNotFound     = errors.New("upload not found")
\tErrLimitReached = errors.New("upload limit reached for this tier")
\tErrInvalidFile  = errors.New("invalid file type")
)

type Service struct {
\tdb  *pgxpool.Pool
\tcfg *config.Config
\ts3  *s3.Client
}

func NewService(db *pgxpool.Pool, cfg *config.Config) (*Service, error) {
\tvar s3Client *s3.Client
\tif cfg.S3Endpoint != "" {
\t\t// Cloudflare R2 or custom endpoint
\t\tawsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
\t\t\tawsconfig.WithRegion(cfg.S3Region),
\t\t\tawsconfig.WithCredentialsProvider(
\t\t\t\tcredentials.NewStaticCredentialsProvider(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, ""),
\t\t\t),
\t\t)
\t\tif err != nil {
\t\t\treturn nil, err
\t\t}
\t\ts3Client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
\t\t\to.BaseEndpoint = aws.String(cfg.S3Endpoint)
\t\t\to.UsePathStyle = true
\t\t})
\t} else {
\t\tawsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
\t\t\tawsconfig.WithRegion(cfg.S3Region),
\t\t\tawsconfig.WithCredentialsProvider(
\t\t\t\tcredentials.NewStaticCredentialsProvider(cfg.S3AccessKeyID, cfg.S3SecretAccessKey, ""),
\t\t\t),
\t\t)
\t\tif err != nil {
\t\t\treturn nil, err
\t\t}
\t\ts3Client = s3.NewFromConfig(awsCfg)
\t}
\treturn &Service{db: db, cfg: cfg, s3: s3Client}, nil
}

// GuestUpload handles a multipart file upload from a guest.
func (s *Service) GuestUpload(ctx context.Context, weddingID string, file multipart.File, header *multipart.FileHeader, guestName string) (*models.Upload, error) {
\t// Validate mime type
\tmimeType := header.Header.Get("Content-Type")
\tfileType, ok := allowedMimeTypes[mimeType]
\tif !ok {
\t\treturn nil, ErrInvalidFile
\t}

\t// Check upload limit
\tvar wedding models.Wedding
\terr := s.db.QueryRow(ctx, "SELECT id, tier FROM weddings WHERE id=$1 AND is_active=true", weddingID).
\t\tScan(&wedding.ID, &wedding.Tier)
\tif errors.Is(err, pgx.ErrNoRows) {
\t\treturn nil, errors.New("wedding not found")
\t}
\tif err != nil {
\t\treturn nil, err
\t}
\tif limit := wedding.UploadLimit(); limit != -1 {
\t\tvar count int
\t\t_ = s.db.QueryRow(ctx, "SELECT COUNT(*) FROM uploads WHERE wedding_id=$1", weddingID).Scan(&count)
\t\tif count >= limit {
\t\t\treturn nil, ErrLimitReached
\t\t}
\t}

\t// Generate S3 key
\text := strings.ToLower(filepath.Ext(header.Filename))
\tfileKey := fmt.Sprintf("weddings/%s/%s%s", weddingID, uuid.NewString(), ext)

\t// Upload to S3/R2
\t_, err = s.s3.PutObject(ctx, &s3.PutObjectInput{
\t\tBucket:      aws.String(s.cfg.S3Bucket),
\t\tKey:         aws.String(fileKey),
\t\tBody:        file,
\t\tContentType: aws.String(mimeType),
\t})
\tif err != nil {
\t\treturn nil, fmt.Errorf("upload to storage: %w", err)
\t}

\tfileURL := fmt.Sprintf("%s/%s", s.cfg.S3PublicBaseURL, fileKey)
\tupload := &models.Upload{
\t\tID:         uuid.NewString(),
\t\tWeddingID:  weddingID,
\t\tFileURL:    fileURL,
\t\tFileKey:    fileKey,
\t\tFileType:   fileType,
\t\tMimeType:   mimeType,
\t\tSizeBytes:  header.Size,
\t\tCategory:   models.CategoryOther, // AI classification is async
\t\tIsApproved: true,
\t\tUploadedAt: time.Now(),
\t}
\tif guestName != "" {
\t\tupload.GuestName = &guestName
\t}

\t_, err = s.db.Exec(ctx, `
\t\tINSERT INTO uploads (id, wedding_id, guest_name, file_url, file_key, file_type, mime_type, size_bytes, category, is_approved)
\t\tVALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
\t\tupload.ID, upload.WeddingID, upload.GuestName, upload.FileURL, upload.FileKey,
\t\tupload.FileType, upload.MimeType, upload.SizeBytes, upload.Category, upload.IsApproved,
\t)
\tif err != nil {
\t\treturn nil, fmt.Errorf("save upload record: %w", err)
\t}
\treturn upload, nil
}

// ListForWedding returns all uploads for a wedding (owner view).
func (s *Service) ListForWedding(ctx context.Context, weddingID string) ([]*models.Upload, error) {
\trows, err := s.db.Query(ctx, `
\t\tSELECT id, wedding_id, guest_name, file_url, file_type, mime_type, size_bytes, category, is_approved, uploaded_at
\t\tFROM uploads WHERE wedding_id=$1 ORDER BY uploaded_at DESC`, weddingID)
\tif err != nil {
\t\treturn nil, err
\t}
\tdefer rows.Close()
\tvar uploads []*models.Upload
\tfor rows.Next() {
\t\tu := &models.Upload{}
\t\tif err := rows.Scan(&u.ID, &u.WeddingID, &u.GuestName, &u.FileURL, &u.FileType, &u.MimeType,
\t\t\t&u.SizeBytes, &u.Category, &u.IsApproved, &u.UploadedAt); err != nil {
\t\t\treturn nil, err
\t\t}
\t\tuploads = append(uploads, u)
\t}
\treturn uploads, nil
}

// SetApproval approves or rejects an upload.
func (s *Service) SetApproval(ctx context.Context, uploadID string, approved bool) error {
\t_, err := s.db.Exec(ctx, "UPDATE uploads SET is_approved=$1 WHERE id=$2", approved, uploadID)
\treturn err
}

// Delete removes an upload record and its S3 object.
func (s *Service) Delete(ctx context.Context, uploadID string) error {
\tvar fileKey string
\terr := s.db.QueryRow(ctx, "SELECT file_key FROM uploads WHERE id=$1", uploadID).Scan(&fileKey)
\tif errors.Is(err, pgx.ErrNoRows) {
\t\treturn ErrNotFound
\t}
\tif err != nil {
\t\treturn err
\t}
\t_, _ = s.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
\t\tBucket: aws.String(s.cfg.S3Bucket),
\t\tKey:    aws.String(fileKey),
\t})
\t_, err = s.db.Exec(ctx, "DELETE FROM uploads WHERE id=$1", uploadID)
\treturn err
}
"""

# ── upload/handler.go ─────────────────────────────────────────────────────────
files["internal/upload/handler.go"] = """\
package upload

import (
\t"encoding/json"
\t"errors"
\t"net/http"

\t"github.com/go-chi/chi/v5"
\t"github.com/storyvows/backend/internal/config"
\t"github.com/storyvows/backend/internal/models"
\t"github.com/storyvows/backend/internal/realtime"
)

type Handler struct {
\tsvc *Service
\tcfg *config.Config
\thub *realtime.Hub
}

func NewHandler(svc *Service, cfg *config.Config, hub *realtime.Hub) *Handler {
\treturn &Handler{svc: svc, cfg: cfg, hub: hub}
}

// GuestUpload handles unauthenticated guest photo/video uploads.
func (h *Handler) GuestUpload(w http.ResponseWriter, r *http.Request) {
\tr.Body = http.MaxBytesReader(w, r.Body, h.cfg.MaxUploadSize)
\tif err := r.ParseMultipartForm(h.cfg.MaxUploadSize); err != nil {
\t\twriteError(w, http.StatusRequestEntityTooLarge, "file too large")
\t\treturn
\t}

\tslug := chi.URLParam(r, "slug")
\tguestName := r.FormValue("guest_name")

\t// Resolve wedding ID from slug
\tweddingID := r.FormValue("wedding_id") // passed by client or resolved here
\tif weddingID == "" {
\t\twriteError(w, http.StatusBadRequest, "wedding_id required")
\t\treturn
\t}
\t_ = slug // slug used for public page routing

\tfile, header, err := r.FormFile("file")
\tif err != nil {
\t\twriteError(w, http.StatusBadRequest, "file field required")
\t\treturn
\t}
\tdefer file.Close()

\tupload, err := h.svc.GuestUpload(r.Context(), weddingID, file, header, guestName)
\tif errors.Is(err, ErrLimitReached) {
\t\twriteError(w, http.StatusPaymentRequired, err.Error())
\t\treturn
\t}
\tif errors.Is(err, ErrInvalidFile) {
\t\twriteError(w, http.StatusUnsupportedMediaType, err.Error())
\t\treturn
\t}
\tif err != nil {
\t\twriteError(w, http.StatusInternalServerError, "upload failed")
\t\treturn
\t}

\t// Broadcast to live wall
\th.hub.Broadcast(weddingID, upload)

\twriteJSON(w, http.StatusCreated, upload)
}

// List returns all uploads for a wedding (owner only, auth required).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
\tuploads, err := h.svc.ListForWedding(r.Context(), chi.URLParam(r, "id"))
\tif err != nil {
\t\twriteError(w, http.StatusInternalServerError, "failed to list uploads")
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, uploads)
}

// Approve sets the is_approved flag on an upload.
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
\tvar req models.ApproveUploadRequest
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid request body")
\t\treturn
\t}
\tif err := h.svc.SetApproval(r.Context(), chi.URLParam(r, "uploadId"), req.Approved); err != nil {
\t\twriteError(w, http.StatusInternalServerError, "failed to update approval")
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, models.SuccessResponse{Message: "updated"})
}

// Delete removes an upload.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
\tif err := h.svc.Delete(r.Context(), chi.URLParam(r, "uploadId")); err != nil {
\t\twriteError(w, http.StatusInternalServerError, "delete failed")
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, models.SuccessResponse{Message: "deleted"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(status)
\t_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
\twriteJSON(w, status, models.ErrorResponse{Error: msg})
}
"""

# ── realtime/hub.go ───────────────────────────────────────────────────────────
files["internal/realtime/hub.go"] = """\
package realtime

import (
\t"encoding/json"
\t"fmt"
\t"net/http"
\t"sync"

\t"github.com/go-chi/chi/v5"
)

// Hub manages SSE connections grouped by wedding ID.
type Hub struct {
\tmu      sync.RWMutex
\tclients map[string][]chan []byte // weddingID → list of client channels
}

func NewHub() *Hub {
\treturn &Hub{clients: make(map[string][]chan []byte)}
}

// Broadcast sends an event to all connected clients for a wedding.
func (h *Hub) Broadcast(weddingID string, payload any) {
\tdata, err := json.Marshal(payload)
\tif err != nil {
\t\treturn
\t}
\tmsg := []byte(fmt.Sprintf("data: %s\\n\\n", data))

\th.mu.RLock()
\tdefer h.mu.RUnlock()
\tfor _, ch := range h.clients[weddingID] {
\t\tselect {
\t\tcase ch <- msg:
\t\tdefault: // skip if client is too slow
\t\t}
\t}
}

// ServeSSE is the HTTP handler for the live wall SSE endpoint.
func (h *Hub) ServeSSE(w http.ResponseWriter, r *http.Request) {
\tweddingID := chi.URLParam(r, "id")

\tw.Header().Set("Content-Type", "text/event-stream")
\tw.Header().Set("Cache-Control", "no-cache")
\tw.Header().Set("Connection", "keep-alive")
\tw.Header().Set("X-Accel-Buffering", "no")

\tch := make(chan []byte, 16)
\th.subscribe(weddingID, ch)
\tdefer h.unsubscribe(weddingID, ch)

\t// Send a connected heartbeat
\t_, _ = fmt.Fprintf(w, "event: connected\\ndata: {}\\n\\n")
\tif f, ok := w.(http.Flusher); ok {
\t\tf.Flush()
\t}

\tfor {
\t\tselect {
\t\tcase msg := <-ch:
\t\t\t_, _ = w.Write(msg)
\t\t\tif f, ok := w.(http.Flusher); ok {
\t\t\t\tf.Flush()
\t\t\t}
\t\tcase <-r.Context().Done():
\t\t\treturn
\t\t}
\t}
}

func (h *Hub) subscribe(weddingID string, ch chan []byte) {
\th.mu.Lock()
\tdefer h.mu.Unlock()
\th.clients[weddingID] = append(h.clients[weddingID], ch)
}

func (h *Hub) unsubscribe(weddingID string, ch chan []byte) {
\th.mu.Lock()
\tdefer h.mu.Unlock()
\tlist := h.clients[weddingID]
\tfor i, c := range list {
\t\tif c == ch {
\t\t\th.clients[weddingID] = append(list[:i], list[i+1:]...)
\t\t\tbreak
\t\t}
\t}
}
"""

# ── gallery/handler.go ────────────────────────────────────────────────────────
files["internal/gallery/handler.go"] = """\
package gallery

import (
\t"archive/zip"
\t"encoding/json"
\t"fmt"
\t"io"
\t"net/http"
\t"net/url"
\t"time"

\t"github.com/go-chi/chi/v5"
\t"github.com/jackc/pgx/v5/pgxpool"
\t"github.com/storyvows/backend/internal/models"
)

type Handler struct {
\tdb *pgxpool.Pool
}

func NewHandler(db *pgxpool.Pool) *Handler {
\treturn &Handler{db: db}
}

// Album returns all approved uploads for a wedding, grouped by category.
func (h *Handler) Album(w http.ResponseWriter, r *http.Request) {
\tweddingID := chi.URLParam(r, "id")
\trows, err := h.db.Query(r.Context(), `
\t\tSELECT id, wedding_id, guest_name, file_url, file_type, mime_type, size_bytes, category, is_approved, uploaded_at
\t\tFROM uploads
\t\tWHERE wedding_id=$1 AND is_approved=true
\t\tORDER BY uploaded_at ASC`, weddingID)
\tif err != nil {
\t\twriteError(w, http.StatusInternalServerError, "failed to load album")
\t\treturn
\t}
\tdefer rows.Close()

\talbum := map[string][]*models.Upload{}
\tfor rows.Next() {
\t\tu := &models.Upload{}
\t\tif err := rows.Scan(&u.ID, &u.WeddingID, &u.GuestName, &u.FileURL, &u.FileType, &u.MimeType,
\t\t\t&u.SizeBytes, &u.Category, &u.IsApproved, &u.UploadedAt); err != nil {
\t\t\tcontinue
\t\t}
\t\talbum[string(u.Category)] = append(album[string(u.Category)], u)
\t}
\twriteJSON(w, http.StatusOK, album)
}

// Highlights returns a curated set of photos (stubbed — AI integration point).
func (h *Handler) Highlights(w http.ResponseWriter, r *http.Request) {
\tweddingID := chi.URLParam(r, "id")
\trows, err := h.db.Query(r.Context(), `
\t\tSELECT id, wedding_id, guest_name, file_url, file_type, mime_type, size_bytes, category, is_approved, uploaded_at
\t\tFROM uploads
\t\tWHERE wedding_id=$1 AND is_approved=true AND file_type='photo'
\t\tORDER BY RANDOM() LIMIT 20`, weddingID)
\tif err != nil {
\t\twriteError(w, http.StatusInternalServerError, "failed to load highlights")
\t\treturn
\t}
\tdefer rows.Close()

\tvar highlights []*models.Upload
\tfor rows.Next() {
\t\tu := &models.Upload{}
\t\t_ = rows.Scan(&u.ID, &u.WeddingID, &u.GuestName, &u.FileURL, &u.FileType,
\t\t\t&u.MimeType, &u.SizeBytes, &u.Category, &u.IsApproved, &u.UploadedAt)
\t\thighlights = append(highlights, u)
\t}
\twriteJSON(w, http.StatusOK, highlights)
}

// Download streams a ZIP of all uploads (gated to Heritage/Legacy tiers).
func (h *Handler) Download(w http.ResponseWriter, r *http.Request) {
\tweddingID := chi.URLParam(r, "id")

\t// Tier check
\tvar tier models.Tier
\t_ = h.db.QueryRow(r.Context(), "SELECT tier FROM weddings WHERE id=$1", weddingID).Scan(&tier)
\tif tier == models.TierElopement {
\t\twriteError(w, http.StatusPaymentRequired, "bulk download requires Heritage or Legacy tier")
\t\treturn
\t}

\trows, err := h.db.Query(r.Context(),
\t\t"SELECT file_url, id FROM uploads WHERE wedding_id=$1 AND is_approved=true", weddingID)
\tif err != nil {
\t\twriteError(w, http.StatusInternalServerError, "query failed")
\t\treturn
\t}
\tdefer rows.Close()

\tw.Header().Set("Content-Type", "application/zip")
\tw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\\"%s-album.zip\\"", weddingID))
\tzw := zip.NewWriter(w)
\tdefer zw.Close()

\tclient := &http.Client{Timeout: 30 * time.Second}
\tfor rows.Next() {
\t\tvar fileURL, id string
\t\t_ = rows.Scan(&fileURL, &id)
\t\tresp, err := client.Get(fileURL)
\t\tif err != nil {
\t\t\tcontinue
\t\t}
\t\tparsed, _ := url.Parse(fileURL)
\t\tfileName := fmt.Sprintf("%s%s", id, parsedExtension(parsed.Path))
\t\tf, err := zw.Create(fileName)
\t\tif err != nil {
\t\t\tresp.Body.Close()
\t\t\tcontinue
\t\t}
\t\t_, _ = io.Copy(f, resp.Body)
\t\tresp.Body.Close()
\t}
}

func parsedExtension(path string) string {
\tfor i := len(path) - 1; i >= 0; i-- {
\t\tif path[i] == '.' {
\t\t\treturn path[i:]
\t\t}
\t\tif path[i] == '/' {
\t\t\tbreak
\t\t}
\t}
\treturn ""
}

func writeJSON(w http.ResponseWriter, status int, data any) {
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(status)
\t_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
\twriteJSON(w, status, models.ErrorResponse{Error: msg})
}
"""

# ── payment/service.go ────────────────────────────────────────────────────────
files["internal/payment/service.go"] = """\
package payment

import (
\t"context"
\t"errors"
\t"fmt"
\t"time"

\t"github.com/google/uuid"
\t"github.com/jackc/pgx/v5/pgxpool"
\t"github.com/storyvows/backend/internal/config"
\t"github.com/storyvows/backend/internal/models"
\t"github.com/stripe/stripe-go/v82"
\t"github.com/stripe/stripe-go/v82/checkout/session"
)

type Service struct {
\tdb  *pgxpool.Pool
\tcfg *config.Config
}

func NewService(db *pgxpool.Pool, cfg *config.Config) *Service {
\tstripe.Key = cfg.StripeSecretKey
\treturn &Service{db: db, cfg: cfg}
}

// CreateCheckout creates a Stripe Checkout session for the requested tier.
func (s *Service) CreateCheckout(ctx context.Context, userID string, req models.CheckoutRequest) (*models.CheckoutResponse, error) {
\tamount, name := s.tierDetails(req.Tier)
\tif amount == 0 {
\t\treturn nil, errors.New("invalid tier")
\t}

\tparams := &stripe.CheckoutSessionParams{
\t\tMode: stripe.String(string(stripe.CheckoutSessionModePayment)),
\t\tLineItems: []*stripe.CheckoutSessionLineItemParams{
\t\t\t{
\t\t\t\tPriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
\t\t\t\t\tCurrency: stripe.String("usd"),
\t\t\t\t\tUnitAmount: stripe.Int64(amount),
\t\t\t\t\tProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
\t\t\t\t\t\tName: stripe.String(fmt.Sprintf("Story Vows — %s", name)),
\t\t\t\t\t},
\t\t\t\t},
\t\t\t\tQuantity: stripe.Int64(1),
\t\t\t},
\t\t},
\t\tSuccessURL: stripe.String(fmt.Sprintf("%s/dashboard?payment=success", s.cfg.FrontendURL)),
\t\tCancelURL:  stripe.String(fmt.Sprintf("%s/pricing?payment=cancelled", s.cfg.FrontendURL)),
\t\tMetadata: map[string]string{
\t\t\t"user_id":    userID,
\t\t\t"wedding_id": req.WeddingID,
\t\t\t"tier":       string(req.Tier),
\t\t},
\t}

\tsess, err := session.New(params)
\tif err != nil {
\t\treturn nil, fmt.Errorf("create stripe session: %w", err)
\t}

\t// Persist pending order
\t_, err = s.db.Exec(ctx, `
\t\tINSERT INTO orders (id, wedding_id, user_id, tier, amount_cents, currency, status, stripe_session_id)
\t\tVALUES ($1,$2,$3,$4,$5,'usd','pending',$6)`,
\t\tuuid.NewString(), req.WeddingID, userID, req.Tier, amount, sess.ID,
\t)
\tif err != nil {
\t\treturn nil, fmt.Errorf("save order: %w", err)
\t}

\treturn &models.CheckoutResponse{
\t\tCheckoutURL: sess.URL,
\t\tSessionID:   sess.ID,
\t}, nil
}

// HandleWebhook processes a Stripe webhook event and activates the correct tier.
func (s *Service) HandleWebhook(ctx context.Context, event stripe.Event) error {
\tif event.Type != "checkout.session.completed" {
\t\treturn nil
\t}

\tvar sess stripe.CheckoutSession
\tif err := event.DataObjectFor(&sess); err != nil {
\t\treturn fmt.Errorf("parse session: %w", err)
\t}

\tweddingID := sess.Metadata["wedding_id"]
\ttier := models.Tier(sess.Metadata["tier"])
\tpaymentIntentID := ""
\tif sess.PaymentIntent != nil {
\t\tpaymentIntentID = sess.PaymentIntent.ID
\t}

\t// Mark order as paid
\t_, err := s.db.Exec(ctx, `
\t\tUPDATE orders
\t\tSET status='paid', stripe_payment_intent_id=$1, paid_at=NOW()
\t\tWHERE stripe_session_id=$2`,
\t\tpaymentIntentID, sess.ID,
\t)
\tif err != nil {
\t\treturn fmt.Errorf("update order: %w", err)
\t}

\t// Set expiry for Elopement tier (1 year)
\tvar expiresAt *time.Time
\tif tier == models.TierElopement {
\t\tt := time.Now().AddDate(1, 0, 0)
\t\texpiresAt = &t
\t}

\t// Activate tier on wedding
\t_, err = s.db.Exec(ctx,
\t\t"UPDATE weddings SET tier=$1, expires_at=$2, updated_at=NOW() WHERE id=$3",
\t\ttier, expiresAt, weddingID,
\t)
\treturn err
}

func (s *Service) tierDetails(tier models.Tier) (int64, string) {
\tswitch tier {
\tcase models.TierElopement:
\t\treturn s.cfg.StripeElopementPrice, "Elopement"
\tcase models.TierHeritage:
\t\treturn s.cfg.StripeHeritagePrice, "Heritage"
\tcase models.TierLegacy:
\t\treturn s.cfg.StripeLegacyPrice, "Legacy"
\t}
\treturn 0, ""
}
"""

# ── payment/handler.go ────────────────────────────────────────────────────────
files["internal/payment/handler.go"] = """\
package payment

import (
\t"encoding/json"
\t"io"
\t"net/http"

\t"github.com/storyvows/backend/internal/config"
\t"github.com/storyvows/backend/internal/middleware"
\t"github.com/storyvows/backend/internal/models"
\t"github.com/stripe/stripe-go/v82"
\t"github.com/stripe/stripe-go/v82/webhook"
)

type Handler struct {
\tsvc *Service
\tcfg *config.Config
}

func NewHandler(svc *Service, cfg *config.Config) *Handler {
\treturn &Handler{svc: svc, cfg: cfg}
}

func (h *Handler) Checkout(w http.ResponseWriter, r *http.Request) {
\tvar req models.CheckoutRequest
\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid request body")
\t\treturn
\t}
\tresp, err := h.svc.CreateCheckout(r.Context(), middleware.GetUserID(r), req)
\tif err != nil {
\t\twriteError(w, http.StatusInternalServerError, err.Error())
\t\treturn
\t}
\twriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) StripeWebhook(w http.ResponseWriter, r *http.Request) {
\tbody, err := io.ReadAll(io.LimitReader(r.Body, 65536))
\tif err != nil {
\t\twriteError(w, http.StatusBadRequest, "read body failed")
\t\treturn
\t}
\tevent, err := webhook.ConstructEvent(body, r.Header.Get("Stripe-Signature"), h.cfg.StripeWebhookSecret)
\tif err != nil {
\t\twriteError(w, http.StatusBadRequest, "invalid stripe signature")
\t\treturn
\t}
\tif err := h.svc.HandleWebhook(r.Context(), event); err != nil {
\t\twriteError(w, http.StatusInternalServerError, "webhook processing failed")
\t\treturn
\t}
\tw.WriteHeader(http.StatusOK)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(status)
\t_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
\twriteJSON(w, status, models.ErrorResponse{Error: msg})
}
"""

# ── main.go ───────────────────────────────────────────────────────────────────
files["cmd/api/main.go"] = """\
package main

import (
\t"context"
\t"fmt"
\t"log/slog"
\t"net/http"
\t"os"
\t"os/signal"
\t"syscall"
\t"time"

\t"github.com/go-chi/chi/v5"
\tchiMiddleware "github.com/go-chi/chi/v5/middleware"
\t"github.com/go-chi/cors"
\t"github.com/go-chi/httprate"
\t"github.com/storyvows/backend/internal/auth"
\t"github.com/storyvows/backend/internal/config"
\t"github.com/storyvows/backend/internal/db"
\t"github.com/storyvows/backend/internal/gallery"
\tappMiddleware "github.com/storyvows/backend/internal/middleware"
\t"github.com/storyvows/backend/internal/payment"
\t"github.com/storyvows/backend/internal/realtime"
\t"github.com/storyvows/backend/internal/upload"
\t"github.com/storyvows/backend/internal/wedding"
)

func main() {
\tlogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
\tslog.SetDefault(logger)

\tcfg, err := config.Load()
\tif err != nil {
\t\tslog.Error("failed to load config", "error", err)
\t\tos.Exit(1)
\t}

\tctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
\tdefer cancel()

\tpool, err := db.New(ctx, cfg.DatabaseURL)
\tif err != nil {
\t\tslog.Error("failed to connect to database", "error", err)
\t\tos.Exit(1)
\t}
\tdefer pool.Close()

\t// ── Services ──────────────────────────────────────────────────────────────
\tauthSvc := auth.NewService(pool, cfg)
\tweddingSvc := wedding.NewService(pool, cfg)
\tuploadSvc, err := upload.NewService(pool, cfg)
\tif err != nil {
\t\tslog.Error("failed to init upload service", "error", err)
\t\tos.Exit(1)
\t}
\tpaymentSvc := payment.NewService(pool, cfg)
\thub := realtime.NewHub()

\t// ── Handlers ──────────────────────────────────────────────────────────────
\tauthHandler := auth.NewHandler(authSvc)
\tweddingHandler := wedding.NewHandler(weddingSvc)
\tuploadHandler := upload.NewHandler(uploadSvc, cfg, hub)
\tpaymentHandler := payment.NewHandler(paymentSvc, cfg)
\tgalleryHandler := gallery.NewHandler(pool)

\t// ── Router ────────────────────────────────────────────────────────────────
\tr := chi.NewRouter()

\tr.Use(chiMiddleware.Recoverer)
\tr.Use(appMiddleware.Logger)
\tr.Use(cors.Handler(cors.Options{
\t\tAllowedOrigins:   []string{cfg.FrontendURL},
\t\tAllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
\t\tAllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
\t\tAllowCredentials: true,
\t\tMaxAge:           300,
\t}))
\tr.Use(httprate.LimitAll(200, time.Minute)) // 200 req/min global rate limit

\tr.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
\t\tw.Header().Set("Content-Type", "application/json")
\t\t_, _ = fmt.Fprintf(w, `{"status":"ok"}`)
\t})

\t// ── Auth routes ───────────────────────────────────────────────────────────
\tr.Route("/api/auth", func(r chi.Router) {
\t\tr.Post("/signup", authHandler.SignUp)
\t\tr.Post("/signin", authHandler.SignIn)
\t\tr.Post("/refresh", authHandler.Refresh)
\t\tr.With(appMiddleware.RequireAuth(cfg.JWTSecret)).Post("/signout", authHandler.SignOut)
\t\tr.With(appMiddleware.RequireAuth(cfg.JWTSecret)).Get("/me", authHandler.Me)
\t})

\t// ── Authenticated couple routes ───────────────────────────────────────────
\tr.Route("/api/weddings", func(r chi.Router) {
\t\tr.Use(appMiddleware.RequireAuth(cfg.JWTSecret))
\t\tr.Post("/", weddingHandler.Create)
\t\tr.Get("/", weddingHandler.List)
\t\tr.Get("/{id}", weddingHandler.Get)
\t\tr.Patch("/{id}", weddingHandler.Update)
\t\tr.Delete("/{id}", weddingHandler.Delete)
\t\tr.Patch("/{id}/privacy", weddingHandler.SetPrivacy)
\t\t// Uploads management (couple)
\t\tr.Get("/{id}/uploads", uploadHandler.List)
\t\tr.Patch("/{id}/uploads/{uploadId}/approve", uploadHandler.Approve)
\t\tr.Delete("/{id}/uploads/{uploadId}", uploadHandler.Delete)
\t\t// Gallery & album
\t\tr.Get("/{id}/album", galleryHandler.Album)
\t\tr.Get("/{id}/album/highlights", galleryHandler.Highlights)
\t\tr.Get("/{id}/album/download", galleryHandler.Download)
\t\t// Live wall SSE
\t\tr.Get("/{id}/wall", hub.ServeSSE)
\t})

\t// ── Guest routes (no auth) ────────────────────────────────────────────────
\tr.Route("/api/w/{slug}", func(r chi.Router) {
\t\tr.Get("/", weddingHandler.GuestView)
\t\tr.Post("/access", weddingHandler.GuestAccess)
\t\tr.Post("/uploads", uploadHandler.GuestUpload)
\t})

\t// ── Payment routes ────────────────────────────────────────────────────────
\tr.With(appMiddleware.RequireAuth(cfg.JWTSecret)).Post("/api/checkout", paymentHandler.Checkout)
\tr.Post("/api/webhooks/stripe", paymentHandler.StripeWebhook)

\t// ── Server ────────────────────────────────────────────────────────────────
\tsrv := &http.Server{
\t\tAddr:         ":" + cfg.Port,
\t\tHandler:      r,
\t\tReadTimeout:  15 * time.Second,
\t\tWriteTimeout: 30 * time.Second,
\t\tIdleTimeout:  120 * time.Second,
\t}

\tslog.Info("starting server", "port", cfg.Port, "env", cfg.Env)
\tgo func() {
\t\tif err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
\t\t\tslog.Error("server error", "error", err)
\t\t\tos.Exit(1)
\t\t}
\t}()

\tquit := make(chan os.Signal, 1)
\tsignal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
\t<-quit
\tslog.Info("shutting down...")

\tshutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
\tdefer shutCancel()
\t_ = srv.Shutdown(shutCtx)
\tslog.Info("shutdown complete")
}
"""

# ── migrations ────────────────────────────────────────────────────────────────
files["migrations/001_init.up.sql"] = """\
-- Enable uuid extension
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Refresh tokens
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);

-- Weddings
CREATE TABLE IF NOT EXISTS weddings (
    id              TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    owner_id        TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_names    TEXT NOT NULL,
    wedding_date    DATE NOT NULL,
    venue           TEXT NOT NULL DEFAULT '',
    welcome_message TEXT NOT NULL DEFAULT '',
    qr_slug         TEXT NOT NULL UNIQUE,
    qr_code_url     TEXT NOT NULL DEFAULT '',
    tier            TEXT NOT NULL DEFAULT 'elopement' CHECK (tier IN ('elopement','heritage','legacy')),
    privacy         TEXT NOT NULL DEFAULT 'public' CHECK (privacy IN ('public','private','password_protected')),
    password_hash   TEXT,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_weddings_owner_id ON weddings(owner_id);
CREATE INDEX IF NOT EXISTS idx_weddings_qr_slug  ON weddings(qr_slug);

-- Uploads
CREATE TABLE IF NOT EXISTS uploads (
    id          TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    wedding_id  TEXT NOT NULL REFERENCES weddings(id) ON DELETE CASCADE,
    guest_name  TEXT,
    file_url    TEXT NOT NULL,
    file_key    TEXT NOT NULL,
    file_type   TEXT NOT NULL CHECK (file_type IN ('photo','video')),
    mime_type   TEXT NOT NULL,
    size_bytes  BIGINT NOT NULL DEFAULT 0,
    category    TEXT NOT NULL DEFAULT 'other' CHECK (category IN ('ceremony','candid','dancing','family','other')),
    is_approved BOOLEAN NOT NULL DEFAULT TRUE,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_uploads_wedding_id ON uploads(wedding_id);
CREATE INDEX IF NOT EXISTS idx_uploads_category   ON uploads(category);

-- Orders
CREATE TABLE IF NOT EXISTS orders (
    id                       TEXT PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    wedding_id               TEXT NOT NULL REFERENCES weddings(id) ON DELETE CASCADE,
    user_id                  TEXT NOT NULL REFERENCES users(id),
    tier                     TEXT NOT NULL CHECK (tier IN ('elopement','heritage','legacy')),
    amount_cents             BIGINT NOT NULL,
    currency                 TEXT NOT NULL DEFAULT 'usd',
    status                   TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','paid','refunded')),
    stripe_session_id        TEXT NOT NULL UNIQUE,
    stripe_payment_intent_id TEXT,
    paid_at                  TIMESTAMPTZ,
    expires_at               TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_orders_wedding_id ON orders(wedding_id);
CREATE INDEX IF NOT EXISTS idx_orders_user_id    ON orders(user_id);
"""

files["migrations/001_init.down.sql"] = """\
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS uploads;
DROP TABLE IF EXISTS weddings;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;
"""

# ── .env.example ──────────────────────────────────────────────────────────────
files[".env.example"] = """\
# Server
PORT=8080
ENV=development

# Database (PostgreSQL)
DATABASE_URL=postgres://user:password@localhost:5432/storyvows?sslmode=disable

# JWT
JWT_SECRET=change-me-to-a-long-random-secret
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=720h

# Stripe (one-time payments, no subscriptions)
STRIPE_SECRET_KEY=sk_test_...
STRIPE_WEBHOOK_SECRET=whsec_...
# Prices in cents: Elopement $199, Heritage $449, Legacy $799
STRIPE_ELOPEMENT_PRICE=19900
STRIPE_HERITAGE_PRICE=44900
STRIPE_LEGACY_PRICE=79900

# Cloudflare R2 (S3-compatible) — or use AWS S3 (leave S3_ENDPOINT empty)
S3_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com
S3_BUCKET=storyvows-uploads
S3_REGION=auto
S3_ACCESS_KEY_ID=...
S3_SECRET_ACCESS_KEY=...
S3_PUBLIC_BASE_URL=https://uploads.yourdomain.com

# App
FRONTEND_URL=http://localhost:3000
MAX_UPLOAD_SIZE=52428800
"""

# Write all files
for rel_path, content in files.items():
    abs_path = os.path.join(BASE, rel_path)
    os.makedirs(os.path.dirname(abs_path), exist_ok=True)
    with open(abs_path, 'w') as f:
        f.write(content)
    print(f"wrote {rel_path}")

print("\\nAll files written successfully.")
