package dto

import "time"

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

type UploadAnalysis struct {
	Status        AnalysisStatus   `json:"status" bson:"status"`
	Category      UploadCategory   `json:"category" bson:"category"`
	SceneTags     []string         `json:"scene_tags,omitempty" bson:"scene_tags,omitempty"`
	DetectedFaces *int             `json:"detected_faces,omitempty" bson:"detected_faces,omitempty"`
	QualityScore  *float64         `json:"quality_score,omitempty" bson:"quality_score,omitempty"`
	EmotionScore  *float64         `json:"emotion_score,omitempty" bson:"emotion_score,omitempty"`
	FeaturedScore *int             `json:"featured_score,omitempty" bson:"featured_score,omitempty"`
	SafeScore     *float64         `json:"safe_score,omitempty" bson:"safe_score,omitempty"`
	Error         *string          `json:"error,omitempty" bson:"error,omitempty"`
	Processing    ProcessingStages `json:"processing,omitempty" bson:"processing,omitempty"`
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
	IsApproved     bool             `json:"is_approved" bson:"is_approved"`
	UploadedAt     time.Time        `json:"uploaded_at" bson:"uploaded_at"`
	Storage        UploadStorage    `json:"storage,omitempty" bson:"storage,omitempty"`
	Metadata       UploadMetadata   `json:"metadata,omitempty" bson:"metadata,omitempty"`
	Timeline       UploadTimeline   `json:"timeline,omitempty" bson:"timeline,omitempty"`
	Analysis       UploadAnalysis   `json:"analysis,omitempty" bson:"analysis,omitempty"`
	Grouping       UploadGrouping   `json:"grouping,omitempty" bson:"grouping,omitempty"`
	Moderation     UploadModeration `json:"moderation,omitempty" bson:"moderation,omitempty"`
}

type ApproveUploadRequest struct {
	Approved bool `json:"approved"`
}
