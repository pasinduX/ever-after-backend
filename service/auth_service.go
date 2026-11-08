package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/resend/resend-go/v3"
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

const (
	passwordResetTTL         = 15 * time.Minute
	passwordResetCooldown    = time.Minute
	passwordResetMaxAttempts = 5
	minPasswordLength        = 8
)

// RequestPasswordReset emails a 6-digit reset code. It reports success for
// unknown emails too, so the endpoint can't be used to discover which
// addresses have accounts.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return nil
	}
	user, err := dao.FindUserByEmail(ctx, s.db, email)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("auth RequestPasswordReset find: %w", err)
	}

	now := time.Now()
	// Throttle resends so the form can't be used to spam someone's inbox.
	if exp := user.PasswordResetExpiresAt; exp != nil && now.Before(exp.Add(-passwordResetTTL+passwordResetCooldown)) {
		return nil
	}

	code := functions.GenerateOTP()
	user.PasswordResetCodeHash = functions.HashToken(code)
	user.PasswordResetExpiresAt = ptrTime(now.Add(passwordResetTTL))
	user.PasswordResetAttempts = 0
	if err := dao.UpdateUser(ctx, s.db, user); err != nil {
		return fmt.Errorf("auth RequestPasswordReset update: %w", err)
	}

	if err := s.sendEmail(
		user.Email,
		"Reset your MoonVeils password",
		fmt.Sprintf("Your password reset code is %s. It expires in 15 minutes. If you didn't ask to reset your password, you can ignore this email.", code),
		fmt.Sprintf("<p>Your password reset code is <strong>%s</strong>. It expires in 15 minutes.</p><p>If you didn't ask to reset your password, you can ignore this email.</p>", code),
	); err != nil {
		return fmt.Errorf("auth RequestPasswordReset email: %w", err)
	}
	return nil
}

// ResetPassword checks the emailed code, sets the new password and signs the
// user in. Every existing session is revoked, since a reset usually means the
// old password can't be trusted.
func (s *AuthService) ResetPassword(ctx context.Context, req dto.ResetPasswordRequest) (*dto.AuthResponse, error) {
	if len(req.NewPassword) < minPasswordLength {
		return nil, apperrors.ErrWeakPassword
	}

	user, err := dao.ClaimPasswordResetAttempt(ctx, s.db, strings.TrimSpace(req.Email), passwordResetMaxAttempts)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return nil, apperrors.ErrInvalidResetCode
		}
		return nil, fmt.Errorf("auth ResetPassword claim: %w", err)
	}

	expired := user.PasswordResetExpiresAt == nil || time.Now().After(*user.PasswordResetExpiresAt)
	matches := subtle.ConstantTimeCompare(
		[]byte(functions.HashToken(strings.TrimSpace(req.Code))),
		[]byte(user.PasswordResetCodeHash),
	) == 1
	if expired || !matches {
		return nil, apperrors.ErrInvalidResetCode
	}

	hash, err := functions.HashPassword(req.NewPassword)
	if err != nil {
		return nil, fmt.Errorf("auth ResetPassword hash: %w", err)
	}
	user.PasswordHash = hash
	user.PasswordResetCodeHash = ""
	user.PasswordResetExpiresAt = nil
	user.PasswordResetAttempts = 0
	// Receiving the code proves the user owns the address.
	user.IsEmailVerified = true
	user.EmailVerificationCode = ""
	user.VerificationExpiresAt = nil
	if err := dao.UpdateUser(ctx, s.db, user); err != nil {
		return nil, fmt.Errorf("auth ResetPassword update: %w", err)
	}
	if err := dao.DeleteRefreshTokensByUser(ctx, s.db, user.ID); err != nil {
		return nil, fmt.Errorf("auth ResetPassword revoke sessions: %w", err)
	}
	return s.issueTokens(ctx, user)
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
	return s.sendEmail(
		email,
		"Verify your MoonVeils account",
		fmt.Sprintf("Your verification code is %s. It expires in 15 minutes.", code),
		fmt.Sprintf("<p>Your verification code is <strong>%s</strong>. It expires in 15 minutes.</p>", code),
	)
}

func (s *AuthService) sendEmail(email, subject, text, html string) error {
	client := resend.NewClient(s.cfg.ResendAPIKey)

	params := &resend.SendEmailRequest{
		From:    s.cfg.ResendFromEmail,
		To:      []string{email},
		Subject: subject,
		Html:    html,
		Text:    text,
	}

	sent, err := client.Emails.Send(params)
	if err != nil {
		slog.Error("resend send failed", "email", email, "subject", subject, "error", err)
		return fmt.Errorf("resend error: %w", err)
	}

	slog.Info("email sent", "email", email, "subject", subject, "messageID", sent.Id)
	return nil
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
