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

type Upload struct {
	ID         string         `json:"id" bson:"_id,omitempty"`
	WeddingID  string         `json:"wedding_id" bson:"wedding_id"`
	GuestName  *string        `json:"guest_name,omitempty" bson:"guest_name,omitempty"`
	FileURL    string         `json:"file_url" bson:"file_url"`
	FileKey    string         `json:"-" bson:"file_key"`
	FileType   FileType       `json:"file_type" bson:"file_type"`
	MimeType   string         `json:"mime_type" bson:"mime_type"`
	SizeBytes  int64          `json:"size_bytes" bson:"size_bytes"`
	Category   UploadCategory `json:"category" bson:"category"`
	IsApproved bool           `json:"is_approved" bson:"is_approved"`
	UploadedAt time.Time      `json:"uploaded_at" bson:"uploaded_at"`
}

type ApproveUploadRequest struct {
	Approved bool `json:"approved"`
}
