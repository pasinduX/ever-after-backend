package dto

import "time"

type AlbumPageStatus string

const (
	// AlbumPageGenerating: the art director is writing this version.
	AlbumPageGenerating AlbumPageStatus = "generating"
	// AlbumPageReady: the page passed every check and can be shown.
	AlbumPageReady AlbumPageStatus = "ready"
	// AlbumPageFallback: no usable page; clients show the block album.
	AlbumPageFallback AlbumPageStatus = "fallback"
)

// Fallback reasons the app can explain to the couple.
const (
	AlbumPageNoPhotos       = "no_photos"
	AlbumPageNotConfigured  = "ai_not_configured"
	AlbumPageQuotaExhausted = "ai_quota_exhausted"
	AlbumPageUnavailable    = "model_unavailable"
	AlbumPageInvalid        = "model_page_invalid"
	AlbumPageInterrupted    = "interrupted"
)

// AlbumPage is one AI-designed album page for a wedding. Every generation is
// kept as a new version; the newest version is the one shown.
type AlbumPage struct {
	ID        string          `json:"id" bson:"_id"`
	WeddingID string          `json:"wedding_id" bson:"wedding_id"`
	Version   int             `json:"version" bson:"version"`
	Status    AlbumPageStatus `json:"status" bson:"status"`
	// Active marks the version guests see: the newest ready design, unless
	// the couple restored an earlier one.
	Active          bool        `json:"active" bson:"active"`
	HTML            string      `json:"html,omitempty" bson:"html,omitempty"`
	FallbackReason  string      `json:"fallback_reason,omitempty" bson:"fallback_reason,omitempty"`
	Palette         []PageColor `json:"palette,omitempty" bson:"palette,omitempty"`
	DisplayFont     string      `json:"display_font,omitempty" bson:"display_font,omitempty"`
	BodyFont        string      `json:"body_font,omitempty" bson:"body_font,omitempty"`
	DesignRationale string      `json:"design_rationale,omitempty" bson:"design_rationale,omitempty"`
	Attempts        int         `json:"attempts" bson:"attempts"`
	// Diagnostics records why attempts were rejected; kept for debugging,
	// never sent to clients.
	Diagnostics  []string  `json:"-" bson:"diagnostics,omitempty"`
	Model        string    `json:"-" bson:"model,omitempty"`
	InputTokens  int       `json:"-" bson:"input_tokens,omitempty"`
	OutputTokens int       `json:"-" bson:"output_tokens,omitempty"`
	LatencyMS    int64     `json:"-" bson:"latency_ms,omitempty"`
	CreatedAt    time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" bson:"updated_at"`
}

type PageColor struct {
	Role string `json:"role" bson:"role"`
	Hex  string `json:"hex" bson:"hex"`
}
