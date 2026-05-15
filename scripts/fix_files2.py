#!/usr/bin/env python3
"""Rewrites remaining corrupt Go files."""
import os

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

def w(rel_path, content):
    full = os.path.join(ROOT, rel_path)
    os.makedirs(os.path.dirname(full), exist_ok=True)
    with open(full, 'w') as f:
        f.write(content)
    print(f"OK {rel_path}")

# ─── dao/dao_create_user.go ───────────────────────────────────────────────
w("dao/dao_create_user.go", """\
package dao

import (
\t"context"
\t"fmt"

\t"github.com/jackc/pgx/v5/pgxpool"
\t"github.com/storyvows/backend/dto"
)

// CreateUser inserts a new user record into the database.
func CreateUser(ctx context.Context, db *pgxpool.Pool, user *dto.User) error {
\t_, err := db.Exec(ctx,
\t\t`INSERT INTO users (id, email, password_hash, full_name) VALUES ($1, $2, $3, $4)`,
\t\tuser.ID, user.Email, user.PasswordHash, user.FullName,
\t)
\tif err != nil {
\t\treturn fmt.Errorf("dao CreateUser: %w", err)
\t}
\treturn nil
}
""")

# ─── api/api_auth_user.go ─────────────────────────────────────────────────
w("api/api_auth_user.go", """\
package api

import (
\t"encoding/json"
\t"errors"
\t"net/http"

\tappErrors "github.com/storyvows/backend/errors"
\t"github.com/storyvows/backend/dto"
\t"github.com/storyvows/backend/service"
\t"github.com/storyvows/backend/utils"
)

func SignUp(svc *service.AuthService) http.HandlerFunc {
\treturn func(w http.ResponseWriter, r *http.Request) {
\t\tvar req dto.SignUpRequest
\t\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\t\tutils.SendErrorResponse(w, http.StatusBadRequest, "invalid request body")
\t\t\treturn
\t\t}
\t\tresp, err := svc.SignUp(r.Context(), req)
\t\tif err != nil {
\t\t\tif errors.Is(err, appErrors.ErrEmailTaken) {
\t\t\t\tutils.SendErrorResponse(w, http.StatusConflict, "email already in use")
\t\t\t\treturn
\t\t\t}
\t\t\tutils.SendErrorResponse(w, http.StatusInternalServerError, "failed to create account")
\t\t\treturn
\t\t}
\t\tutils.SendJSON(w, http.StatusCreated, resp)
\t}
}

func SignIn(svc *service.AuthService) http.HandlerFunc {
\treturn func(w http.ResponseWriter, r *http.Request) {
\t\tvar req dto.SignInRequest
\t\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\t\tutils.SendErrorResponse(w, http.StatusBadRequest, "invalid request body")
\t\t\treturn
\t\t}
\t\tresp, err := svc.SignIn(r.Context(), req)
\t\tif err != nil {
\t\t\tif errors.Is(err, appErrors.ErrInvalidCreds) {
\t\t\t\tutils.SendErrorResponse(w, http.StatusUnauthorized, "invalid email or password")
\t\t\t\treturn
\t\t\t}
\t\t\tutils.SendErrorResponse(w, http.StatusInternalServerError, "failed to sign in")
\t\t\treturn
\t\t}
\t\tutils.SendJSON(w, http.StatusOK, resp)
\t}
}

func Refresh(svc *service.AuthService) http.HandlerFunc {
\treturn func(w http.ResponseWriter, r *http.Request) {
\t\tvar req dto.RefreshRequest
\t\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\t\tutils.SendErrorResponse(w, http.StatusBadRequest, "invalid request body")
\t\t\treturn
\t\t}
\t\tresp, err := svc.RefreshTokens(r.Context(), req.RefreshToken)
\t\tif err != nil {
\t\t\tif errors.Is(err, appErrors.ErrInvalidToken) {
\t\t\t\tutils.SendErrorResponse(w, http.StatusUnauthorized, "invalid or expired refresh token")
\t\t\t\treturn
\t\t\t}
\t\t\tutils.SendErrorResponse(w, http.StatusInternalServerError, "failed to refresh tokens")
\t\t\treturn
\t\t}
\t\tutils.SendJSON(w, http.StatusOK, resp)
\t}
}

func SignOut(svc *service.AuthService) http.HandlerFunc {
\treturn func(w http.ResponseWriter, r *http.Request) {
\t\tvar req dto.RefreshRequest
\t\tif err := json.NewDecoder(r.Body).Decode(&req); err != nil {
\t\t\tutils.SendErrorResponse(w, http.StatusBadRequest, "invalid request body")
\t\t\treturn
\t\t}
\t\tif err := svc.SignOut(r.Context(), req.RefreshToken); err != nil {
\t\t\tutils.SendErrorResponse(w, http.StatusInternalServerError, "failed to sign out")
\t\t\treturn
\t\t}
\t\tutils.SendSuccessResponse(w, "signed out successfully", nil)
\t}
}

func Me(svc *service.AuthService, getUID func(*http.Request) string) http.HandlerFunc {
\treturn func(w http.ResponseWriter, r *http.Request) {
\t\tuserID := getUID(r)
\t\tuser, err := svc.Me(r.Context(), userID)
\t\tif err != nil {
\t\t\tutils.SendErrorResponse(w, http.StatusInternalServerError, "failed to fetch user")
\t\t\treturn
\t\t}
\t\tutils.SendJSON(w, http.StatusOK, user)
\t}
}
""")

# ─── service/auth_service.go ──────────────────────────────────────────────
w("service/auth_service.go", """\
package service

import (
\t"context"
\t"errors"
\t"fmt"
\t"time"

\t"github.com/golang-jwt/jwt/v5"
\t"github.com/google/uuid"
\t"github.com/jackc/pgx/v5/pgxpool"
\t"github.com/storyvows/backend/dao"
\t"github.com/storyvows/backend/dto"
\tappErrors "github.com/storyvows/backend/errors"
\t"github.com/storyvows/backend/functions"
\t"github.com/storyvows/backend/integrations"
)

type AuthService struct {
\tdb  *pgxpool.Pool
\tcfg *integrations.Secrets
}

func NewAuthService(db *pgxpool.Pool, cfg *integrations.Secrets) *AuthService {
\treturn &AuthService{db: db, cfg: cfg}
}

func (s *AuthService) SignUp(ctx context.Context, req dto.SignUpRequest) (*dto.AuthResponse, error) {
\tcount, err := dao.CountUsersByEmail(ctx, s.db, req.Email)
\tif err != nil {
\t\treturn nil, fmt.Errorf("auth SignUp count: %w", err)
\t}
\tif count > 0 {
\t\treturn nil, appErrors.ErrEmailTaken
\t}
\thash, err := functions.HashPassword(req.Password)
\tif err != nil {
\t\treturn nil, fmt.Errorf("auth SignUp hash: %w", err)
\t}
\tuser := &dto.User{
\t\tID:           uuid.NewString(),
\t\tEmail:        req.Email,
\t\tPasswordHash: hash,
\t\tFullName:     req.FullName,
\t}
\tif err := dao.CreateUser(ctx, s.db, user); err != nil {
\t\treturn nil, fmt.Errorf("auth SignUp create: %w", err)
\t}
\treturn s.issueTokens(ctx, user)
}

func (s *AuthService) SignIn(ctx context.Context, req dto.SignInRequest) (*dto.AuthResponse, error) {
\tuser, err := dao.FindUserByEmail(ctx, s.db, req.Email)
\tif err != nil {
\t\tif errors.Is(err, dao.ErrNoRows) {
\t\t\treturn nil, appErrors.ErrInvalidCreds
\t\t}
\t\treturn nil, fmt.Errorf("auth SignIn find: %w", err)
\t}
\tif !functions.CheckPassword(req.Password, user.PasswordHash) {
\t\treturn nil, appErrors.ErrInvalidCreds
\t}
\treturn s.issueTokens(ctx, user)
}

func (s *AuthService) RefreshTokens(ctx context.Context, rawToken string) (*dto.AuthResponse, error) {
\thash := functions.HashToken(rawToken)
\trt, err := dao.FindRefreshTokenByHash(ctx, s.db, hash)
\tif err != nil {
\t\tif errors.Is(err, dao.ErrNoRows) {
\t\t\treturn nil, appErrors.ErrInvalidToken
\t\t}
\t\treturn nil, fmt.Errorf("auth Refresh find: %w", err)
\t}
\tif time.Now().After(rt.ExpiresAt) {
\t\t_ = dao.DeleteRefreshTokenByID(ctx, s.db, rt.ID)
\t\treturn nil, appErrors.ErrInvalidToken
\t}
\t_ = dao.DeleteRefreshTokenByID(ctx, s.db, rt.ID)
\tuser, err := dao.FindUserByID(ctx, s.db, rt.UserID)
\tif err != nil {
\t\treturn nil, fmt.Errorf("auth Refresh find user: %w", err)
\t}
\treturn s.issueTokens(ctx, user)
}

func (s *AuthService) SignOut(ctx context.Context, rawToken string) error {
\thash := functions.HashToken(rawToken)
\tif err := dao.DeleteRefreshTokenByHash(ctx, s.db, hash); err != nil {
\t\treturn fmt.Errorf("auth SignOut delete: %w", err)
\t}
\treturn nil
}

func (s *AuthService) Me(ctx context.Context, userID string) (*dto.User, error) {
\tuser, err := dao.FindUserByID(ctx, s.db, userID)
\tif err != nil {
\t\treturn nil, fmt.Errorf("auth Me find: %w", err)
\t}
\treturn user, nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *dto.User) (*dto.AuthResponse, error) {
\tnow := time.Now()
\taccessClaims := jwt.MapClaims{
\t\t"sub": user.ID,
\t\t"exp": now.Add(s.cfg.JWTAccessTokenTTL).Unix(),
\t\t"iat": now.Unix(),
\t}
\taccessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(s.cfg.JWTSecret))
\tif err != nil {
\t\treturn nil, fmt.Errorf("sign access token: %w", err)
\t}
\trawRefresh := uuid.NewString()
\trt := &dto.RefreshToken{
\t\tID:        uuid.NewString(),
\t\tUserID:    user.ID,
\t\tTokenHash: functions.HashToken(rawRefresh),
\t\tExpiresAt: now.Add(s.cfg.JWTRefreshTokenTTL),
\t}
\tif err := dao.CreateRefreshToken(ctx, s.db, rt); err != nil {
\t\treturn nil, fmt.Errorf("store refresh token: %w", err)
\t}
\treturn &dto.AuthResponse{
\t\tAccessToken:  accessToken,
\t\tRefreshToken: rawRefresh,
\t\tUser:         user,
\t}, nil
}
""")

print("All files written successfully.")
