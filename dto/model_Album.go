package dto

import "time"

type ActLabel string

const (
	ActAnticipation ActLabel = "ANTICIPATION"
	ActCeremony     ActLabel = "CEREMONY"
	ActFamilyBonds  ActLabel = "FAMILY_BONDS"
	ActJustTheTwo   ActLabel = "JUST_THE_TWO"
	ActCelebration  ActLabel = "CELEBRATION"
	ActFinalDance   ActLabel = "FINAL_DANCE"
)

type AlbumStyle string

const (
	AlbumStyleCinematic AlbumStyle = "cinematic"
	AlbumStyleRomantic  AlbumStyle = "romantic"
)

type AlbumGenerationStatus string

const (
	AlbumStatusPending    AlbumGenerationStatus = "pending"
	AlbumStatusProcessing AlbumGenerationStatus = "processing"
	AlbumStatusCompleted  AlbumGenerationStatus = "completed"
	AlbumStatusFailed     AlbumGenerationStatus = "failed"
)

type StoryAct struct {
	ID          string    `json:"id" bson:"_id,omitempty"`
	WeddingID   string    `json:"wedding_id" bson:"wedding_id"`
	Label       ActLabel  `json:"label" bson:"label"`
	Title       string    `json:"title" bson:"title"`
	Quote       string    `json:"quote" bson:"quote"`
	Confidence  float64   `json:"confidence" bson:"confidence"`
	NeedsReview bool      `json:"needs_review" bson:"needs_review"`
	Confirmed   bool      `json:"confirmed" bson:"confirmed"`
	PhotoIDs    []string  `json:"photo_ids" bson:"photo_ids"`
	HeroPhotoID string    `json:"hero_photo_id" bson:"hero_photo_id"`
	Order       int       `json:"order" bson:"order"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" bson:"updated_at"`
}

type AlbumConfig struct {
	ID          string                `json:"id" bson:"_id,omitempty"`
	WeddingID   string                `json:"wedding_id" bson:"wedding_id"`
	Style       AlbumStyle            `json:"style" bson:"style"`
	Status      AlbumGenerationStatus `json:"status" bson:"status"`
	ActsCount   int                   `json:"acts_count" bson:"acts_count"`
	Confidence  float64               `json:"confidence" bson:"confidence"`
	NeedsReview bool                  `json:"needs_review" bson:"needs_review"`
	Error       *string               `json:"error,omitempty" bson:"error,omitempty"`
	UpdatedAt   time.Time             `json:"updated_at" bson:"updated_at"`
}

type GenerateAlbumRequest struct {
	Style AlbumStyle `json:"style"`
}

type ActConfirmation struct {
	ID        string   `json:"id"`
	PhotoIDs  []string `json:"photo_ids"`
	Confirmed bool     `json:"confirmed"`
}

type ConfirmActsRequest struct {
	Acts []ActConfirmation `json:"acts"`
}

type SetAlbumStyleRequest struct {
	Style AlbumStyle `json:"style"`
}

// --- Album payload (blocks-based render format) ---

type AlbumTheme struct {
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	Font           string `json:"font"`
	Mood           string `json:"mood"`
}

type AlbumCoverBlock struct {
	ImageID  string `json:"image_id"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

type BlockOverlay struct {
	Title     string `json:"title,omitempty"`
	Alignment string `json:"alignment,omitempty"`
}

type BlockRightContent struct {
	Quote string `json:"quote,omitempty"`
}

type AlbumBlock struct {
	ID              string             `json:"id"`
	Type            string             `json:"type"`
	ImageID         string             `json:"image_id,omitempty"`
	Images          []string           `json:"images,omitempty"`
	VideoID         string             `json:"video_id,omitempty"`
	Text            string             `json:"text,omitempty"`
	Title           string             `json:"title,omitempty"`
	Subtitle        string             `json:"subtitle,omitempty"`
	Caption         string             `json:"caption,omitempty"`
	Layout          string             `json:"layout,omitempty"`
	Height          string             `json:"height,omitempty"`
	AspectRatio     string             `json:"aspect_ratio,omitempty"`
	Columns         int                `json:"columns,omitempty"`
	BackgroundImage string             `json:"background_image,omitempty"`
	LeftImage       string             `json:"left_image,omitempty"`
	RightContent    *BlockRightContent `json:"right_content,omitempty"`
	Overlay         *BlockOverlay      `json:"overlay,omitempty"`
}

type AlbumPayload struct {
	AlbumID string          `json:"album_id"`
	Style   AlbumStyle      `json:"style"`
	Theme   AlbumTheme      `json:"theme"`
	Cover   AlbumCoverBlock `json:"cover"`
	Blocks  []AlbumBlock    `json:"blocks"`
}
