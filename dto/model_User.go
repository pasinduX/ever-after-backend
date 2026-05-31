package dto

import "time"

type User struct {
	ID                    string     `json:"id" bson:"_id,omitempty"`
	Email                 string     `json:"email" bson:"email"`
	PasswordHash          string     `json:"-" bson:"password_hash"`
	FullName              string     `json:"full_name" bson:"full_name"`
	IsEmailVerified       bool       `json:"is_email_verified" bson:"is_email_verified"`
	EmailVerificationCode string     `json:"-" bson:"email_verification_code,omitempty"`
	VerificationExpiresAt *time.Time `json:"-" bson:"verification_expires_at,omitempty"`
	CreatedAt             time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" bson:"updated_at"`
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

type SendWhatsAppRequest struct {
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
}

type SendWhatsAppTemplateRequest struct {
	PhoneNumber      string            `json:"phone_number"`
	ContentSid       string            `json:"content_sid"`
	ContentVariables map[string]string `json:"content_variables"`
}

type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	User         *User  `json:"user"`
}
