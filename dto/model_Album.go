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
	PrimaryColor   string `json:"primary_color" bson:"primary_color"`
	SecondaryColor string `json:"secondary_color" bson:"secondary_color"`
	Font           string `json:"font" bson:"font"`
	Mood           string `json:"mood" bson:"mood"`
}

// AssetObject replaces bare image ID strings so the renderer has everything
// it needs (URL, dimensions) without additional fetches.
type AssetObject struct {
	ID          string  `json:"id" bson:"id"`
	URL         string  `json:"url" bson:"url"`
	Width       int     `json:"width,omitempty" bson:"width,omitempty"`
	Height      int     `json:"height,omitempty" bson:"height,omitempty"`
	AspectRatio float64 `json:"aspect_ratio,omitempty" bson:"aspect_ratio,omitempty"`
	Alt         string  `json:"alt,omitempty" bson:"alt,omitempty"`
}

type AlbumCoverBlock struct {
	Image    *AssetObject `json:"image" bson:"image"`
	Title    string       `json:"title" bson:"title"`
	Subtitle string       `json:"subtitle" bson:"subtitle"`
}

type BlockOverlay struct {
	Title       string  `json:"title,omitempty" bson:"title,omitempty"`
	Alignment   string  `json:"alignment,omitempty" bson:"alignment,omitempty"`
	RevealDelay float64 `json:"reveal_delay,omitempty" bson:"reveal_delay,omitempty"` // seconds before text fades in
}

type BlockRightContent struct {
	Quote string `json:"quote,omitempty" bson:"quote,omitempty"`
}

// BlockMotion carries pacing/transition hints for the renderer.
type BlockMotion struct {
	Reveal   string  `json:"reveal,omitempty" bson:"reveal,omitempty"`     // fade-in | fade-up | slide-left
	Parallax float64 `json:"parallax,omitempty" bson:"parallax,omitempty"` // 0.0–1.0 scroll depth factor
	KenBurns bool    `json:"ken_burns,omitempty" bson:"ken_burns,omitempty"`
	Duration float64 `json:"duration,omitempty" bson:"duration,omitempty"` // animation seconds
}

// BlockFilter applies a cinematic colour grade to a single block.
type BlockFilter struct {
	Grade    string  `json:"grade,omitempty" bson:"grade,omitempty"`       // warm | cool | bw | muted
	Vignette float64 `json:"vignette,omitempty" bson:"vignette,omitempty"` // 0.0–1.0
}

type AlbumBlock struct {
	ID              string             `json:"id" bson:"id"`
	Type            string             `json:"type" bson:"type"`
	Image           *AssetObject       `json:"image,omitempty" bson:"image,omitempty"`
	FallbackImage   *AssetObject       `json:"fallback_image,omitempty" bson:"fallback_image,omitempty"`
	Images          []AssetObject      `json:"images,omitempty" bson:"images,omitempty"`
	BackgroundImage *AssetObject       `json:"background_image,omitempty" bson:"background_image,omitempty"`
	LeftImage       *AssetObject       `json:"left_image,omitempty" bson:"left_image,omitempty"`
	Text            string             `json:"text,omitempty" bson:"text,omitempty"`
	Title           string             `json:"title,omitempty" bson:"title,omitempty"`
	Subtitle        string             `json:"subtitle,omitempty" bson:"subtitle,omitempty"`
	Caption         string             `json:"caption,omitempty" bson:"caption,omitempty"`
	Layout          string             `json:"layout,omitempty" bson:"layout,omitempty"`
	Height          string             `json:"height,omitempty" bson:"height,omitempty"`
	Columns         int                `json:"columns,omitempty" bson:"columns,omitempty"`
	RightContent    *BlockRightContent `json:"right_content,omitempty" bson:"right_content,omitempty"`
	Overlay         *BlockOverlay      `json:"overlay,omitempty" bson:"overlay,omitempty"`
	Motion          *BlockMotion       `json:"motion,omitempty" bson:"motion,omitempty"`
	Filter          *BlockFilter       `json:"filter,omitempty" bson:"filter,omitempty"`
}

// AlbumChapter groups blocks under a named story act section.
type AlbumChapter struct {
	ID       string       `json:"id" bson:"id"`
	Title    string       `json:"title" bson:"title"`
	Subtitle string       `json:"subtitle,omitempty" bson:"subtitle,omitempty"`
	Act      ActLabel     `json:"act" bson:"act"`
	Blocks   []AlbumBlock `json:"blocks" bson:"blocks"`
}

// AlbumSoundtrack is an optional ambient audio track for the album.
type AlbumSoundtrack struct {
	URL      string  `json:"url,omitempty" bson:"url,omitempty"`
	Autoplay bool    `json:"autoplay" bson:"autoplay"`
	Volume   float64 `json:"volume" bson:"volume"`
}

type AlbumPayload struct {
	Version    string          `json:"version" bson:"version"`
	AlbumID    string          `json:"album_id" bson:"album_id"`
	Style      AlbumStyle      `json:"style" bson:"style"`
	Theme      AlbumTheme      `json:"theme" bson:"theme"`
	Cover      AlbumCoverBlock `json:"cover" bson:"cover"`
	Chapters   []AlbumChapter  `json:"chapters" bson:"chapters"`
	Soundtrack *AlbumSoundtrack `json:"soundtrack,omitempty" bson:"soundtrack,omitempty"`
}
