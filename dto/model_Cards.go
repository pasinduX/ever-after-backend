package dto

import "time"

type ThankYouTemplate string

const (
	ThankYouTemplateIvorySymphony ThankYouTemplate = "ivory-symphony"
	ThankYouTemplateGoldenHour    ThankYouTemplate = "golden-hour"
	ThankYouTemplateMaisonLumiere ThankYouTemplate = "maison-lumiere"
)

type ThankYouConfig struct {
	ID        string           `json:"id" bson:"_id,omitempty"`
	WeddingID string           `json:"wedding_id" bson:"wedding_id"`
	Template  ThankYouTemplate `json:"template" bson:"template"`
	Couple    string           `json:"couple" bson:"couple"`
	Date      string           `json:"date" bson:"date"`
	Venue     string           `json:"venue" bson:"venue"`
	Hashtag   string           `json:"hashtag" bson:"hashtag"`
	HeroImage string           `json:"hero_image" bson:"hero_image"`
	Portrait  string           `json:"portrait" bson:"portrait"`
	Intro     []string         `json:"intro" bson:"intro"`
	Message   string           `json:"message" bson:"message"`
	Signature string           `json:"signature" bson:"signature"`
	Gallery   []string         `json:"gallery" bson:"gallery"`
	Closing   string           `json:"closing" bson:"closing"`
	CreatedAt time.Time        `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time        `json:"updated_at" bson:"updated_at"`
}

type InviteIntro struct {
	Lines   []string `json:"lines" bson:"lines"`
	Tagline string   `json:"tagline" bson:"tagline"`
	BGImage string   `json:"bg_image" bson:"bg_image"`
}

type InviteStoryEvent struct {
	Year string `json:"year" bson:"year"`
	Text string `json:"text" bson:"text"`
}

type InviteStory struct {
	Title  string             `json:"title" bson:"title"`
	Events []InviteStoryEvent `json:"events" bson:"events"`
}

type InviteDetails struct {
	Date    string `json:"date" bson:"date"`
	Time    string `json:"time" bson:"time"`
	Venue   string `json:"venue" bson:"venue"`
	Address string `json:"address" bson:"address"`
	Dress   string `json:"dress" bson:"dress"`
}

type InviteCountdown struct {
	TargetISO string `json:"target_iso" bson:"target_iso"`
	Label     string `json:"label" bson:"label"`
}

type InviteQR struct {
	Title    string `json:"title" bson:"title"`
	Subtitle string `json:"subtitle" bson:"subtitle"`
	URL      string `json:"url" bson:"url"`
}

type InviteOutro struct {
	Line      string `json:"line" bson:"line"`
	Signature string `json:"signature" bson:"signature"`
}

type InviteConfig struct {
	ID        string          `json:"id" bson:"_id,omitempty"`
	WeddingID string          `json:"wedding_id" bson:"wedding_id"`
	Couple    string          `json:"couple" bson:"couple"`
	Hashtag   string          `json:"hashtag" bson:"hashtag"`
	Intro     InviteIntro     `json:"intro" bson:"intro"`
	Story     InviteStory     `json:"story" bson:"story"`
	Details   InviteDetails   `json:"details" bson:"details"`
	Countdown InviteCountdown `json:"countdown" bson:"countdown"`
	QR        InviteQR        `json:"qr" bson:"qr"`
	Outro     InviteOutro     `json:"outro" bson:"outro"`
	CreatedAt time.Time       `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" bson:"updated_at"`
}

type CreateInviteRequest struct {
	Couple    string          `json:"couple"`
	Hashtag   string          `json:"hashtag"`
	Intro     InviteIntro     `json:"intro"`
	Story     InviteStory     `json:"story"`
	Details   InviteDetails   `json:"details"`
	Countdown InviteCountdown `json:"countdown"`
	QR        InviteQR        `json:"qr"`
	Outro     InviteOutro     `json:"outro"`
}

type UpdateInviteRequest struct {
	Couple    *string          `json:"couple,omitempty"`
	Hashtag   *string          `json:"hashtag,omitempty"`
	Intro     *InviteIntro     `json:"intro,omitempty"`
	Story     *InviteStory     `json:"story,omitempty"`
	Details   *InviteDetails   `json:"details,omitempty"`
	Countdown *InviteCountdown `json:"countdown,omitempty"`
	QR        *InviteQR        `json:"qr,omitempty"`
	Outro     *InviteOutro     `json:"outro,omitempty"`
}

type CreateThankYouRequest struct {
	Template  ThankYouTemplate `json:"template"`
	Couple    string           `json:"couple"`
	Date      string           `json:"date"`
	Venue     string           `json:"venue"`
	Hashtag   string           `json:"hashtag"`
	HeroImage string           `json:"hero_image"`
	Portrait  string           `json:"portrait"`
	Intro     []string         `json:"intro"`
	Message   string           `json:"message"`
	Signature string           `json:"signature"`
	Gallery   []string         `json:"gallery"`
	Closing   string           `json:"closing"`
}

type UpdateThankYouRequest struct {
	Template  *ThankYouTemplate `json:"template,omitempty"`
	Couple    *string           `json:"couple,omitempty"`
	Date      *string           `json:"date,omitempty"`
	Venue     *string           `json:"venue,omitempty"`
	Hashtag   *string           `json:"hashtag,omitempty"`
	HeroImage *string           `json:"hero_image,omitempty"`
	Portrait  *string           `json:"portrait,omitempty"`
	Intro     []string          `json:"intro,omitempty"`
	Message   *string           `json:"message,omitempty"`
	Signature *string           `json:"signature,omitempty"`
	Gallery   []string          `json:"gallery,omitempty"`
	Closing   *string           `json:"closing,omitempty"`
}
