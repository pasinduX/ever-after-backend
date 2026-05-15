package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
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

func (s *AuthService) SignUp(ctx context.Context, req dto.SignUpRequest) error {
	count, err := dao.CountUsersByEmail(ctx, s.db, req.Email)
	if err != nil {
		return fmt.Errorf("auth SignUp count: %w", err)
	}
	if count > 0 {
		return apperrors.ErrEmailTaken
	}

	hash, err := functions.HashPassword(req.Password)
	if err != nil {
		return fmt.Errorf("auth SignUp hash: %w", err)
	}

	code := functions.GenerateOTP()
	now := time.Now()
	user := &dto.User{
		ID:                    uuid.NewString(),
		Email:                 req.Email,
		PasswordHash:          hash,
		FullName:              req.FullName,
		IsEmailVerified:       false,
		EmailVerificationCode: code,
		VerificationExpiresAt: ptrTime(now.Add(15 * time.Minute)),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := dao.CreateUser(ctx, s.db, user); err != nil {
		if isMongoDuplicateKeyError(err) {
			return apperrors.ErrEmailTaken
		}
		return fmt.Errorf("auth SignUp create: %w", err)
	}
	if err := s.sendVerificationEmail(user.Email, code); err != nil {
		_ = dao.DeleteUserByEmail(ctx, s.db, user.Email)
		return fmt.Errorf("auth SignUp email: %w", err)
	}
	return nil
}

func isMongoDuplicateKeyError(err error) bool {
	var writeErr mongo.WriteException
	if errors.As(err, &writeErr) {
		for _, we := range writeErr.WriteErrors {
			if we.Code == 11000 {
				return true
			}
		}
	}
	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		return cmdErr.Code == 11000
	}
	return false
}

func (s *AuthService) VerifyEmail(ctx context.Context, email, code string) (*dto.AuthResponse, error) {
	user, err := dao.FindUserByEmail(ctx, s.db, email)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return nil, apperrors.ErrInvalidVerificationCode
		}
		return nil, fmt.Errorf("auth VerifyEmail find: %w", err)
	}
	if user.IsEmailVerified {
		return s.issueTokens(ctx, user)
	}
	if user.EmailVerificationCode != code || user.VerificationExpiresAt == nil || time.Now().After(*user.VerificationExpiresAt) {
		return nil, apperrors.ErrInvalidVerificationCode
	}
	user.IsEmailVerified = true
	user.EmailVerificationCode = ""
	user.VerificationExpiresAt = nil
	user.UpdatedAt = time.Now()
	if err := dao.UpdateUser(ctx, s.db, user); err != nil {
		return nil, fmt.Errorf("auth VerifyEmail update: %w", err)
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
	if !functions.CheckPassword(user.PasswordHash, req.Password) {
		return nil, apperrors.ErrInvalidCreds
	}
	if !user.IsEmailVerified {
		return nil, apperrors.ErrEmailNotVerified
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

func (s *AuthService) sendVerificationEmail(email, code string) error {
	from := mail.NewEmail("Story Vows", s.cfg.SendGridFromEmail)
	to := mail.NewEmail("", email)
	subject := "Verify your Story Vows account"
	plainTextContent := fmt.Sprintf("Your verification code is %s. It expires in 15 minutes.", code)
	htmlContent := fmt.Sprintf("<p>Your verification code is <strong>%s</strong>. It expires in 15 minutes.</p>", code)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	request := sendgrid.GetRequest(s.cfg.SendGridAPIKey, "/v3/mail/send", "")
	request.Method = "POST"
	if strings.EqualFold(s.cfg.SendGridDataResidency, "EU") {
		var err error
		request, err = sendgrid.SetDataResidency(request, "eu")
		if err != nil {
			return err
		}
	}
	request.Body = mail.GetRequestBody(message)

	response, err := sendgrid.API(request)
	if err != nil {
		slog.Error("sendgrid send failed", "email", email, "error", err)
		return err
	}
	if response.StatusCode >= 400 {
		slog.Error("sendgrid send failed", "email", email, "status", response.StatusCode, "body", response.Body)
		return fmt.Errorf("sendgrid error: %d", response.StatusCode)
	}
	return nil
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
