package dto

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
)

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

type StringArray []string

type IntArray []int

func (s *StringArray) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	raw := bson.RawValue{Type: t, Value: data}
	switch t {
	case bsontype.String:
		var value string
		if err := raw.Unmarshal(&value); err != nil {
			return err
		}
		*s = parseCoupleNames(value)
		return nil
	case bsontype.Array:
		var arr []string
		if err := raw.Unmarshal(&arr); err != nil {
			return err
		}
		*s = arr
		return nil
	default:
		var str string
		if err := raw.Unmarshal(&str); err == nil {
			*s = parseCoupleNames(str)
			return nil
		}
		return fmt.Errorf("cannot decode couple names from BSON type %v", t)
	}
}

func (s *IntArray) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	raw := bson.RawValue{Type: t, Value: data}
	switch t {
	case bsontype.Array:
		var arr []int
		if err := raw.Unmarshal(&arr); err != nil {
			return err
		}
		*s = arr
		return nil
	case bsontype.String:
		var value string
		if err := raw.Unmarshal(&value); err != nil {
			return err
		}
		*s = parseAges(value)
		return nil
	case bsontype.Int32, bsontype.Int64, bsontype.Double:
		var num int
		if err := raw.Unmarshal(&num); err == nil {
			*s = []int{num}
			return nil
		}
		var f float64
		if err := raw.Unmarshal(&f); err == nil {
			*s = []int{int(f)}
			return nil
		}
		return fmt.Errorf("cannot decode ages from BSON type %v", t)
	default:
		var str string
		if err := raw.Unmarshal(&str); err == nil {
			*s = parseAges(str)
			return nil
		}
		return fmt.Errorf("cannot decode ages from BSON type %v", t)
	}
}

func parseCoupleNames(value string) StringArray {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.Contains(value, "&") {
		parts := strings.Split(value, "&")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	return StringArray{value}
}

func parseAges(value string) IntArray {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == ' '
	})
	ages := make([]int, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			continue
		}
		ages = append(ages, n)
	}
	return ages
}

type Wedding struct {
	ID             string      `json:"id" bson:"_id,omitempty"`
	OwnerID        string      `json:"owner_id" bson:"owner_id"`
	CoupleNames    StringArray `json:"couple_names" bson:"couple_names"`
	WeddingDate    time.Time   `json:"wedding_date" bson:"wedding_date"`
	WeddingTime    string      `json:"wedding_time,omitempty" bson:"wedding_time,omitempty"`
	Venue          string      `json:"venue" bson:"venue"`
	Address        string      `json:"address" bson:"address"`
	WhatsAppNumber string      `json:"whatsapp_number" bson:"whatsapp_number"`
	Ages           IntArray    `json:"ages" bson:"ages"`
	WelcomeMessage string      `json:"welcome_message" bson:"welcome_message"`
	Template       string      `json:"template,omitempty" bson:"template,omitempty"`
	Lighting       string      `json:"lighting,omitempty" bson:"lighting,omitempty"`
	StoryStyle     string      `json:"story_style,omitempty" bson:"story_style,omitempty"`
	CeremonyStyle  string      `json:"ceremony_style,omitempty" bson:"ceremony_style,omitempty"`
	VenueType      string      `json:"venue_type,omitempty" bson:"venue_type,omitempty"`
	WeddingMood    string      `json:"wedding_mood,omitempty" bson:"wedding_mood,omitempty"`
	WeddingTheme   string      `json:"wedding_theme,omitempty" bson:"wedding_theme,omitempty"`
	QRSlug         string      `json:"qr_slug" bson:"qr_slug"`
	QRCodeURL      string      `json:"qr_code_url" bson:"qr_code_url"`
	Tier           Tier        `json:"tier" bson:"tier"`
	Privacy        Privacy     `json:"privacy" bson:"privacy"`
	PasswordHash   *string     `json:"-" bson:"password_hash,omitempty"`
	IsActive       bool        `json:"is_active" bson:"is_active"`
	ExpiresAt      *time.Time  `json:"expires_at,omitempty" bson:"expires_at,omitempty"`
	CreatedAt      time.Time   `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at" bson:"updated_at"`
	UploadCount    int         `json:"upload_count,omitempty" bson:"upload_count,omitempty"`
}

func (w *Wedding) UploadLimit() int {
	if w.Tier == TierElopement {
		return 100
	}
	return -1
}

type CreateWeddingRequest struct {
	CoupleNames    []string `json:"couple_names"`
	WeddingDate    string   `json:"wedding_date"`
	WeddingTime    string   `json:"wedding_time,omitempty"`
	Venue          string   `json:"venue"`
	Address        string   `json:"address"`
	WhatsAppNumber string   `json:"whatsapp_number"`
	Ages           []int    `json:"ages"`
	WelcomeMessage string   `json:"welcome_message"`
	Template       string   `json:"template,omitempty"`
	Lighting       string   `json:"lighting,omitempty"`
	StoryStyle     string   `json:"story_style,omitempty"`
	CeremonyStyle  string   `json:"ceremony_style,omitempty"`
	VenueType      string   `json:"venue_type,omitempty"`
	WeddingMood    string   `json:"wedding_mood,omitempty"`
	WeddingTheme   string   `json:"wedding_theme,omitempty"`
}

type UpdateWeddingRequest struct {
	CoupleNames    *[]string `json:"couple_names"`
	WeddingDate    *string   `json:"wedding_date"`
	WeddingTime    *string   `json:"wedding_time,omitempty"`
	Venue          *string   `json:"venue"`
	Address        *string   `json:"address"`
	WhatsAppNumber *string   `json:"whatsapp_number"`
	Ages           *[]int    `json:"ages"`
	WelcomeMessage *string   `json:"welcome_message"`
	Template       *string   `json:"template,omitempty"`
	Lighting       *string   `json:"lighting,omitempty"`
	StoryStyle     *string   `json:"story_style,omitempty"`
	CeremonyStyle  *string   `json:"ceremony_style,omitempty"`
	VenueType      *string   `json:"venue_type,omitempty"`
	WeddingMood    *string   `json:"wedding_mood,omitempty"`
	WeddingTheme   *string   `json:"wedding_theme,omitempty"`
}

type PrivacyRequest struct {
	Privacy  Privacy `json:"privacy"`
	Password *string `json:"password"`
}

type GuestAccessRequest struct {
	Password string `json:"password"`
}
