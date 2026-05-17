package dto

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
	Address        string     `json:"address" bson:"address"`
	WhatsAppNumber string     `json:"whatsapp_number" bson:"whatsapp_number"`
	Ages           string     `json:"ages" bson:"ages"`
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
	Template       string     `json:"template,omitempty" bson:"template,omitempty"`
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
	Address        string `json:"address"`
	WhatsAppNumber string `json:"whatsapp_number"`
	Ages           string `json:"ages"`
	WelcomeMessage string `json:"welcome_message"`
	Template       string `json:"template,omitempty"`
}

type UpdateWeddingRequest struct {
	CoupleNames    *string `json:"couple_names"`
	WeddingDate    *string `json:"wedding_date"`
	Venue          *string `json:"venue"`
	Address        *string `json:"address"`
	WhatsAppNumber *string `json:"whatsapp_number"`
	Ages           *string `json:"ages"`
	WelcomeMessage *string `json:"welcome_message"`
	Template       *string `json:"template,omitempty"`
}

type PrivacyRequest struct {
	Privacy  Privacy `json:"privacy"`
	Password *string `json:"password"`
}

type GuestAccessRequest struct {
	Password string `json:"password"`
}
