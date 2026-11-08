package dto

import "time"

type UploadEXIF struct {
	CapturedAt  *time.Time `json:"captured_at,omitempty" bson:"captured_at,omitempty"`
	CameraMake  string     `json:"camera_make,omitempty" bson:"camera_make,omitempty"`
	CameraModel string     `json:"camera_model,omitempty" bson:"camera_model,omitempty"`
	Lat         *float64   `json:"lat,omitempty" bson:"lat,omitempty"`
	Lng         *float64   `json:"lng,omitempty" bson:"lng,omitempty"`
	// Orientation is the raw EXIF orientation tag (1–8). Derivatives already
	// have it applied, so clients never need to rotate.
	Orientation int `json:"orientation,omitempty" bson:"orientation,omitempty"`
}

type UploadCategory string

const (
	CategoryCeremony UploadCategory = "ceremony"
	CategoryCandid   UploadCategory = "candid"
	CategoryDancing  UploadCategory = "dancing"
	CategoryFamily   UploadCategory = "family"
	CategoryOther    UploadCategory = "other"
)

type FileType string

const (
	FileTypePhoto FileType = "photo"
	FileTypeVideo FileType = "video"
)

type AnalysisStatus string

const (
	AnalysisStatusPending    AnalysisStatus = "pending"
	AnalysisStatusProcessing AnalysisStatus = "processing"
	AnalysisStatusSucceeded  AnalysisStatus = "succeeded"
	AnalysisStatusFailed     AnalysisStatus = "failed"
)

type UploadStorage struct {
	OriginalURL  string `json:"original_url" bson:"original_url"`
	MediumURL    string `json:"medium_url" bson:"medium_url"`
	ThumbnailURL string `json:"thumbnail_url" bson:"thumbnail_url"`
	FileKey      string `json:"file_key" bson:"file_key"`
}

type UploadMetadata struct {
	MimeType    string   `json:"mime_type" bson:"mime_type"`
	SizeBytes   int64    `json:"size_bytes" bson:"size_bytes"`
	Width       *int     `json:"width,omitempty" bson:"width,omitempty"`
	Height      *int     `json:"height,omitempty" bson:"height,omitempty"`
	AspectRatio *float64 `json:"aspect_ratio,omitempty" bson:"aspect_ratio,omitempty"`
	Orientation *string  `json:"orientation,omitempty" bson:"orientation,omitempty"`
	// Palette and Tone are measured from the pixels, not guessed by the AI,
	// so album colours can be built from what the photo actually looks like.
	Palette []PaletteSwatch `json:"palette,omitempty" bson:"palette,omitempty"`
	Tone    *PhotoTone      `json:"tone,omitempty" bson:"tone,omitempty"`
	// PipelineVersion records which image pipeline produced the derivatives,
	// palette and dimensions, so re-analysis knows which uploads to rebuild.
	PipelineVersion int `json:"pipeline_version,omitempty" bson:"pipeline_version,omitempty"`
}

// PaletteSwatch is one measured colour. Role is "ground" (largest area),
// "accent" (most saturated with real presence), "shadow", "highlight" or "mid".
type PaletteSwatch struct {
	Hex   string  `json:"hex" bson:"hex"`
	Share float64 `json:"share" bson:"share"`
	Role  string  `json:"role" bson:"role"`
}

// PhotoTone summarises how a photo reads overall. Temperature runs -1 (cool)
// to 1 (warm); brightness and saturation run 0–1; Key is low, mid or high.
type PhotoTone struct {
	Temperature float64 `json:"temperature" bson:"temperature"`
	Brightness  float64 `json:"brightness" bson:"brightness"`
	Saturation  float64 `json:"saturation" bson:"saturation"`
	Key         string  `json:"key" bson:"key"`
}

type UploadTimeline struct {
	UploadedAt time.Time  `json:"uploaded_at" bson:"uploaded_at"`
	CapturedAt *time.Time `json:"captured_at,omitempty" bson:"captured_at,omitempty"`
}

type ProcessingStages struct {
	Thumbnail      AnalysisStatus `json:"thumbnail" bson:"thumbnail"`
	AIAnalysis     AnalysisStatus `json:"ai_analysis" bson:"ai_analysis"`
	Moderation     AnalysisStatus `json:"moderation" bson:"moderation"`
	DuplicateCheck AnalysisStatus `json:"duplicate_check" bson:"duplicate_check"`
}

type ActCandidate struct {
	Act        string  `json:"act" bson:"act"`
	Confidence float64 `json:"confidence" bson:"confidence"`
}

type UploadAnalysis struct {
	Status            AnalysisStatus `json:"status" bson:"status"`
	Category          UploadCategory `json:"category" bson:"category"`
	SceneTags         []string       `json:"scene_tags,omitempty" bson:"scene_tags,omitempty"`
	DetectedFaces     *int           `json:"detected_faces,omitempty" bson:"detected_faces,omitempty"`
	QualityScore      *float64       `json:"quality_score,omitempty" bson:"quality_score,omitempty"`
	EmotionScore      *float64       `json:"emotion_score,omitempty" bson:"emotion_score,omitempty"`
	FeaturedScore     *int           `json:"featured_score,omitempty" bson:"featured_score,omitempty"`
	SafeScore         *float64       `json:"safe_score,omitempty" bson:"safe_score,omitempty"`
	StudioProbability *float64       `json:"studio_probability,omitempty" bson:"studio_probability,omitempty"`
	ActCandidates     []ActCandidate `json:"act_candidates,omitempty" bson:"act_candidates,omitempty"`
	// EventType classifies the wedding moment captured in this photo.
	// Values: pre_shoot | getting_ready | ceremony | reception | other
	EventType string `json:"event_type,omitempty" bson:"event_type,omitempty"`
	// ClothColors lists dominant outfit colours visible (e.g. ["white","gold"]).
	ClothColors []string         `json:"cloth_colors,omitempty" bson:"cloth_colors,omitempty"`
	Error       *string          `json:"error,omitempty" bson:"error,omitempty"`
	Processing  ProcessingStages `json:"processing,omitempty" bson:"processing,omitempty"`
	// Album holds what an album designer needs from this photo.
	Album *AlbumPhotoAnalysis `json:"album,omitempty" bson:"album,omitempty"`
}

const (
	AlbumFitKeep  = "keep"
	AlbumFitMaybe = "maybe"
	AlbumFitSkip  = "skip"
)

// AlbumPhotoAnalysis is the album-focused read of one photo: whether it
// belongs in the album, how strong it is, which moment it shows, where to
// crop, and where a title could sit.
type AlbumPhotoAnalysis struct {
	Version        int      `json:"version" bson:"version"`
	Model          string   `json:"model,omitempty" bson:"model,omitempty"`
	Fit            string   `json:"fit" bson:"fit"`
	Issues         []string `json:"issues,omitempty" bson:"issues,omitempty"`
	AestheticScore float64  `json:"aesthetic_score" bson:"aesthetic_score"`
	CoverScore     float64  `json:"cover_score" bson:"cover_score"`
	HeroScore      float64  `json:"hero_score" bson:"hero_score"`
	EmotionScore   float64  `json:"emotion_score" bson:"emotion_score"`
	// Moment is the specific wedding moment, e.g. ring_exchange, first_dance.
	Moment        string     `json:"moment" bson:"moment"`
	StoryPhase    string     `json:"story_phase" bson:"story_phase"`
	CouplePresent bool       `json:"couple_present" bson:"couple_present"`
	PeopleCount   int        `json:"people_count" bson:"people_count"`
	ShotStyle     string     `json:"shot_style" bson:"shot_style"`
	FocalPoint    FocalPoint `json:"focal_point" bson:"focal_point"`
	TextArea      string     `json:"text_area" bson:"text_area"`
	AltText       string     `json:"alt_text" bson:"alt_text"`
	Caption       string     `json:"caption,omitempty" bson:"caption,omitempty"`
	CaptionWorthy bool       `json:"caption_worthy" bson:"caption_worthy"`
	AnalysedAt    time.Time  `json:"analysed_at" bson:"analysed_at"`
}

// FocalPoint is the main subject's centre as fractions of width and height,
// usable directly as CSS object-position.
type FocalPoint struct {
	X float64 `json:"x" bson:"x"`
	Y float64 `json:"y" bson:"y"`
}

type UploadGrouping struct {
	MomentGroupID *string `json:"moment_group_id,omitempty" bson:"moment_group_id,omitempty"`
	DuplicateHash *string `json:"duplicate_hash,omitempty" bson:"duplicate_hash,omitempty"`
}

type UploadModeration struct {
	IsApproved bool    `json:"is_approved" bson:"is_approved"`
	ReviewedBy *string `json:"reviewed_by,omitempty" bson:"reviewed_by,omitempty"`
}

type Upload struct {
	ID             string           `json:"id" bson:"_id,omitempty"`
	WeddingID      string           `json:"wedding_id" bson:"wedding_id"`
	GuestName      *string          `json:"guest_name,omitempty" bson:"guest_name,omitempty"`
	FileURL        string           `json:"file_url" bson:"file_url"`
	FileKey        string           `json:"-" bson:"file_key"`
	FileType       FileType         `json:"file_type" bson:"file_type"`
	MimeType       string           `json:"mime_type" bson:"mime_type"`
	SizeBytes      int64            `json:"size_bytes" bson:"size_bytes"`
	Category       UploadCategory   `json:"category" bson:"category"`
	AnalysisStatus AnalysisStatus   `json:"analysis_status" bson:"analysis_status"`
	QualityScore   *float64         `json:"quality_score,omitempty" bson:"quality_score,omitempty"`
	DetectedFaces  *int             `json:"detected_faces,omitempty" bson:"detected_faces,omitempty"`
	Orientation    *string          `json:"orientation,omitempty" bson:"orientation,omitempty"`
	TakenAt        *time.Time       `json:"taken_at,omitempty" bson:"taken_at,omitempty"`
	Location       *string          `json:"location,omitempty" bson:"location,omitempty"`
	SceneTags      []string         `json:"scene_tags,omitempty" bson:"scene_tags,omitempty"`
	AnalysisError  *string          `json:"analysis_error,omitempty" bson:"analysis_error,omitempty"`
	AIInsights     map[string]any   `json:"ai_insights,omitempty" bson:"ai_insights,omitempty"`
	ContentHash    string           `json:"content_hash,omitempty" bson:"content_hash,omitempty"`
	PHash          string           `json:"phash,omitempty" bson:"phash,omitempty"`
	IsApproved     bool             `json:"is_approved" bson:"is_approved"`
	UploadedAt     time.Time        `json:"uploaded_at" bson:"uploaded_at"`
	Storage        UploadStorage    `json:"storage,omitempty" bson:"storage,omitempty"`
	Metadata       UploadMetadata   `json:"metadata,omitempty" bson:"metadata,omitempty"`
	Timeline       UploadTimeline   `json:"timeline,omitempty" bson:"timeline,omitempty"`
	Analysis       UploadAnalysis   `json:"analysis,omitempty" bson:"analysis,omitempty"`
	EXIF           UploadEXIF       `json:"exif,omitempty" bson:"exif,omitempty"`
	Grouping       UploadGrouping   `json:"grouping,omitempty" bson:"grouping,omitempty"`
	Moderation     UploadModeration `json:"moderation,omitempty" bson:"moderation,omitempty"`
}

type ApproveUploadRequest struct {
	Approved bool `json:"approved"`
}

// AlbumPhoto is one photo as the album page needs it: display and thumbnail
// links freshly signed for this request, and its display dimensions.
type AlbumPhoto struct {
	ID      string `json:"id"`
	Src     string `json:"src"`
	Thumb   string `json:"thumb,omitempty"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Caption string `json:"caption,omitempty"`
	Alt     string `json:"alt,omitempty"`
}
