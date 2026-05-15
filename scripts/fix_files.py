#!/usr/bin/env python3
"""Rewrites all migrated Go files to fix duplicate package declaration corruption."""

import os

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

def w(rel_path, content):
    full = os.path.join(ROOT, rel_path)
    os.makedirs(os.path.dirname(full), exist_ok=True)
    with open(full, 'w') as f:
        f.write(content)
    print(f"OK {rel_path}")

# ─── integrations/secrets.go ───────────────────────────────────────────────
w("integrations/secrets.go", """\
package integrations

import (
\t"fmt"
\t"os"
\t"strconv"
\t"time"

\t"github.com/joho/godotenv"
)

type Secrets struct {
\tPort string
\tEnv  string

\tDatabaseURL string

\tJWTSecret          string
\tJWTAccessTokenTTL  time.Duration
\tJWTRefreshTokenTTL time.Duration

\tStripeSecretKey      string
\tStripeWebhookSecret  string
\tStripeElopementPrice int64
\tStripeHeritagePrice  int64
\tStripeLegacyPrice    int64

\tS3Endpoint        string
\tS3Bucket          string
\tS3Region          string
\tS3AccessKeyID     string
\tS3SecretAccessKey string
\tS3PublicBaseURL   string

\tFrontendURL   string
\tMaxUploadSize int64
}

func Load() (*Secrets, error) {
\t_ = godotenv.Load()
\ts := &Secrets{
\t\tPort:               getEnv("PORT", "8080"),
\t\tEnv:                getEnv("ENV", "development"),
\t\tDatabaseURL:        requireEnv("DATABASE_URL"),
\t\tJWTSecret:          requireEnv("JWT_SECRET"),
\t\tJWTAccessTokenTTL:  parseDuration(getEnv("JWT_ACCESS_TTL", "15m")),
\t\tJWTRefreshTokenTTL: parseDuration(getEnv("JWT_REFRESH_TTL", "720h")),
\t\tStripeSecretKey:      requireEnv("STRIPE_SECRET_KEY"),
\t\tStripeWebhookSecret:  requireEnv("STRIPE_WEBHOOK_SECRET"),
\t\tStripeElopementPrice: parseInt64(getEnv("STRIPE_ELOPEMENT_PRICE", "19900")),
\t\tStripeHeritagePrice:  parseInt64(getEnv("STRIPE_HERITAGE_PRICE", "44900")),
\t\tStripeLegacyPrice:    parseInt64(getEnv("STRIPE_LEGACY_PRICE", "79900")),
\t\tS3Endpoint:        getEnv("S3_ENDPOINT", ""),
\t\tS3Bucket:          requireEnv("S3_BUCKET"),
\t\tS3Region:          getEnv("S3_REGION", "auto"),
\t\tS3AccessKeyID:     requireEnv("S3_ACCESS_KEY_ID"),
\t\tS3SecretAccessKey: requireEnv("S3_SECRET_ACCESS_KEY"),
\t\tS3PublicBaseURL:   requireEnv("S3_PUBLIC_BASE_URL"),
\t\tFrontendURL:   getEnv("FRONTEND_URL", "http://localhost:3000"),
\t\tMaxUploadSize: parseInt64(getEnv("MAX_UPLOAD_SIZE", "52428800")),
\t}
\treturn s, nil
}

func getEnv(key, fallback string) string {
\tif v := os.Getenv(key); v != "" {
\t\treturn v
\t}
\treturn fallback
}

func requireEnv(key string) string {
\tv := os.Getenv(key)
\tif v == "" {
\t\tpanic(fmt.Sprintf("required environment variable %q is not set", key))
\t}
\treturn v
}

func parseDuration(s string) time.Duration {
\td, err := time.ParseDuration(s)
\tif err != nil {
\t\tpanic(fmt.Sprintf("invalid duration %q: %v", s, err))
\t}
\treturn d
}

func parseInt64(s string) int64 {
\tn, err := strconv.ParseInt(s, 10, 64)
\tif err != nil {
\t\treturn 0
\t}
\treturn n
}
""")

# ─── dbConfig/dbConnector.go ──────────────────────────────────────────────
w("dbConfig/dbConnector.go", """\
package dbConfig

import (
\t"context"
\t"fmt"

\t"github.com/jackc/pgx/v5/pgxpool"
)

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
\tcfg, err := pgxpool.ParseConfig(databaseURL)
\tif err != nil {
\t\treturn nil, fmt.Errorf("parse db config: %w", err)
\t}
\tcfg.MaxConns = 20
\tpool, err := pgxpool.NewWithConfig(ctx, cfg)
\tif err != nil {
\t\treturn nil, fmt.Errorf("create pool: %w", err)
\t}
\tif err := pool.Ping(ctx); err != nil {
\t\treturn nil, fmt.Errorf("ping db: %w", err)
\t}
\treturn pool, nil
}
""")

# ─── utils/response.go ────────────────────────────────────────────────────
w("utils/response.go", """\
package utils

import (
\t"encoding/json"
\t"net/http"

\t"github.com/storyvows/backend/dto"
)

func SendJSON(w http.ResponseWriter, status int, data any) {
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(status)
\t_ = json.NewEncoder(w).Encode(data)
}

func SendSuccessResponse(w http.ResponseWriter, message string, data any) {
\tSendJSON(w, http.StatusOK, dto.SuccessResponse{Message: message, Data: data})
}

func SendErrorResponse(w http.ResponseWriter, status int, message string) {
\tSendJSON(w, status, dto.ErrorResponse{Error: message})
}
""")

# ─── dto/model_User.go ────────────────────────────────────────────────────
w("dto/model_User.go", """\
package dto

import "time"

type User struct {
\tID           string    `json:"id" db:"id"`
\tEmail        string    `json:"email" db:"email"`
\tPasswordHash string    `json:"-" db:"password_hash"`
\tFullName     string    `json:"full_name" db:"full_name"`
\tCreatedAt    time.Time `json:"created_at" db:"created_at"`
\tUpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type RefreshToken struct {
\tID        string    `json:"id" db:"id"`
\tUserID    string    `json:"user_id" db:"user_id"`
\tTokenHash string    `json:"-" db:"token_hash"`
\tExpiresAt time.Time `json:"expires_at" db:"expires_at"`
\tCreatedAt time.Time `json:"created_at" db:"created_at"`
}

type SignUpRequest struct {
\tEmail    string `json:"email"`
\tPassword string `json:"password"`
\tFullName string `json:"full_name"`
}

type SignInRequest struct {
\tEmail    string `json:"email"`
\tPassword string `json:"password"`
}

type RefreshRequest struct {
\tRefreshToken string `json:"refresh_token"`
}

type AuthResponse struct {
\tAccessToken  string `json:"access_token"`
\tRefreshToken string `json:"refresh_token"`
\tUser         *User  `json:"user"`
}
""")

# ─── realtime/hub.go ──────────────────────────────────────────────────────
w("realtime/hub.go", """\
package realtime

import (
\t"encoding/json"
\t"fmt"
\t"net/http"
\t"sync"

\t"github.com/go-chi/chi/v5"
)

type Hub struct {
\tmu      sync.RWMutex
\tclients map[string][]chan []byte
}

func NewHub() *Hub {
\treturn &Hub{clients: make(map[string][]chan []byte)}
}

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
\t\tdefault:
\t\t}
\t}
}

func (h *Hub) ServeSSE(w http.ResponseWriter, r *http.Request) {
\tweddingID := chi.URLParam(r, "id")
\tw.Header().Set("Content-Type", "text/event-stream")
\tw.Header().Set("Cache-Control", "no-cache")
\tw.Header().Set("Connection", "keep-alive")
\tw.Header().Set("X-Accel-Buffering", "no")

\tch := make(chan []byte, 16)
\th.subscribe(weddingID, ch)
\tdefer h.unsubscribe(weddingID, ch)

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
""")

# ─── functions/function_ValidateJWT.go ────────────────────────────────────
w("functions/function_ValidateJWT.go", """\
package functions

import (
\t"context"
\t"encoding/json"
\t"log/slog"
\t"net/http"
\t"strings"
\t"time"

\t"github.com/golang-jwt/jwt/v5"
\t"github.com/storyvows/backend/dto"
)

type contextKey string

const UserIDKey contextKey = "user_id"

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

func RequireAuth(jwtSecret string) func(http.Handler) http.Handler {
\treturn func(next http.Handler) http.Handler {
\t\treturn http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
\t\t\tauthHeader := r.Header.Get("Authorization")
\t\t\tif !strings.HasPrefix(authHeader, "Bearer ") {
\t\t\t\twriteUnauthorized(w, "missing or invalid authorization header")
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
\t\t\t\twriteUnauthorized(w, "invalid or expired token")
\t\t\t\treturn
\t\t\t}
\t\t\tclaims, ok := token.Claims.(jwt.MapClaims)
\t\t\tif !ok {
\t\t\t\twriteUnauthorized(w, "invalid token claims")
\t\t\t\treturn
\t\t\t}
\t\t\tuserID, ok := claims["sub"].(string)
\t\t\tif !ok {
\t\t\t\twriteUnauthorized(w, "invalid token subject")
\t\t\t\treturn
\t\t\t}
\t\t\tctx := context.WithValue(r.Context(), UserIDKey, userID)
\t\t\tnext.ServeHTTP(w, r.WithContext(ctx))
\t\t})
\t}
}

func GetUserID(r *http.Request) string {
\tuserID, _ := r.Context().Value(UserIDKey).(string)
\treturn userID
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
\tw.Header().Set("Content-Type", "application/json")
\tw.WriteHeader(http.StatusUnauthorized)
\t_ = json.NewEncoder(w).Encode(dto.ErrorResponse{Error: msg})
}
""")

# ─── apiHandlers/router.go ────────────────────────────────────────────────
w("apiHandlers/router.go", """\
package apiHandlers

import (
\t"fmt"
\t"net/http"
\t"time"

\t"github.com/go-chi/chi/v5"
\tchiMiddleware "github.com/go-chi/chi/v5/middleware"
\t"github.com/go-chi/cors"
\t"github.com/go-chi/httprate"
\t"github.com/jackc/pgx/v5/pgxpool"
\t"github.com/storyvows/backend/api"
\t"github.com/storyvows/backend/functions"
\t"github.com/storyvows/backend/integrations"
\t"github.com/storyvows/backend/realtime"
\t"github.com/storyvows/backend/service"
)

func NewRouter(
\tcfg *integrations.Secrets,
\tdb *pgxpool.Pool,
\tauthSvc *service.AuthService,
\tweddingSvc *service.WeddingService,
\tuploadSvc *service.UploadService,
\tpaymentSvc *service.PaymentService,
\thub *realtime.Hub,
) http.Handler {
\tr := chi.NewRouter()

\tr.Use(chiMiddleware.Recoverer)
\tr.Use(functions.Logger)
\tr.Use(cors.Handler(cors.Options{
\t\tAllowedOrigins:   []string{cfg.FrontendURL},
\t\tAllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
\t\tAllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
\t\tAllowCredentials: true,
\t\tMaxAge:           300,
\t}))
\tr.Use(httprate.LimitAll(200, time.Minute))

\tr.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
\t\tw.Header().Set("Content-Type", "application/json")
\t\t_, _ = fmt.Fprintf(w, `{"status":"ok"}`)
\t})

\trequireAuth := functions.RequireAuth(cfg.JWTSecret)
\tgetUID := functions.GetUserID

\tr.Route("/api/auth", func(r chi.Router) {
\t\tr.Post("/signup", api.SignUp(authSvc))
\t\tr.Post("/signin", api.SignIn(authSvc))
\t\tr.Post("/refresh", api.Refresh(authSvc))
\t\tr.With(requireAuth).Post("/signout", api.SignOut(authSvc))
\t\tr.With(requireAuth).Get("/me", api.Me(authSvc, getUID))
\t})

\tr.Route("/api/weddings", func(r chi.Router) {
\t\tr.Use(requireAuth)
\t\tr.Post("/", api.CreateWedding(weddingSvc, getUID))
\t\tr.Get("/", api.ListWeddings(weddingSvc, getUID))
\t\tr.Get("/{id}", api.GetWedding(weddingSvc, getUID))
\t\tr.Patch("/{id}", api.UpdateWedding(weddingSvc, getUID))
\t\tr.Delete("/{id}", api.DeleteWedding(weddingSvc, getUID))
\t\tr.Patch("/{id}/privacy", api.SetPrivacyWedding(weddingSvc, getUID))
\t\tr.Get("/{id}/uploads", api.ListUploads(uploadSvc))
\t\tr.Patch("/{id}/uploads/{uploadId}/approve", api.ApproveUpload(uploadSvc))
\t\tr.Delete("/{id}/uploads/{uploadId}", api.DeleteUpload(uploadSvc))
\t\tr.Get("/{id}/album", api.Album(db))
\t\tr.Get("/{id}/album/highlights", api.Highlights(db))
\t\tr.Get("/{id}/album/download", api.Download(db))
\t\tr.Get("/{id}/wall", hub.ServeSSE)
\t})

\tr.Route("/api/w/{slug}", func(r chi.Router) {
\t\tr.Get("/", api.GuestViewWedding(weddingSvc))
\t\tr.Post("/access", api.GuestAccessWedding(weddingSvc))
\t\tr.Post("/uploads", api.GuestUpload(uploadSvc, hub, cfg.MaxUploadSize))
\t})

\tr.With(requireAuth).Post("/api/checkout", api.Checkout(paymentSvc, getUID))
\tr.Post("/api/webhooks/stripe", api.StripeWebhook(paymentSvc, cfg))

\treturn r
}
""")

# ─── main.go ──────────────────────────────────────────────────────────────
w("main.go", """\
package main

import (
\t"context"
\t"log/slog"
\t"net/http"
\t"os"
\t"os/signal"
\t"syscall"
\t"time"

\t"github.com/storyvows/backend/apiHandlers"
\t"github.com/storyvows/backend/dbConfig"
\t"github.com/storyvows/backend/integrations"
\t"github.com/storyvows/backend/realtime"
\t"github.com/storyvows/backend/service"
)

func main() {
\tlogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
\tslog.SetDefault(logger)

\tcfg, err := integrations.Load()
\tif err != nil {
\t\tslog.Error("failed to load config", "error", err)
\t\tos.Exit(1)
\t}

\tctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
\tdefer cancel()

\tdb, err := dbConfig.Connect(ctx, cfg.DatabaseURL)
\tif err != nil {
\t\tslog.Error("failed to connect to database", "error", err)
\t\tos.Exit(1)
\t}
\tdefer db.Close()

\tauthSvc := service.NewAuthService(db, cfg)
\tweddingSvc := service.NewWeddingService(db, cfg)
\tuploadSvc, err := service.NewUploadService(db, cfg)
\tif err != nil {
\t\tslog.Error("failed to init upload service", "error", err)
\t\tos.Exit(1)
\t}
\tpaymentSvc := service.NewPaymentService(db, cfg)
\thub := realtime.NewHub()

\trouter := apiHandlers.NewRouter(cfg, db, authSvc, weddingSvc, uploadSvc, paymentSvc, hub)

\tsrv := &http.Server{
\t\tAddr:         ":" + cfg.Port,
\t\tHandler:      router,
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
""")

print("All files written successfully.")
