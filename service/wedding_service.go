package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	apperrors "github.com/storyvows/backend/errors"
	"github.com/storyvows/backend/functions"
	"github.com/storyvows/backend/integrations"
	"go.mongodb.org/mongo-driver/mongo"
)

type WeddingService struct {
	db  *mongo.Database
	cfg *integrations.Secrets
	// uploads is used to clear stored objects on delete. Optional: a nil value
	// simply skips object cleanup.
	uploads *UploadService
}

func NewWeddingService(db *mongo.Database, cfg *integrations.Secrets, uploads *UploadService) *WeddingService {
	return &WeddingService{db: db, cfg: cfg, uploads: uploads}
}

func (s *WeddingService) Create(ctx context.Context, ownerID string, req dto.CreateWeddingRequest) (*dto.Wedding, error) {
	// Check if user already has an active wedding
	existingWeddings, err := dao.FindWeddingsByOwner(ctx, s.db, ownerID)
	if err != nil && !errors.Is(err, dao.ErrNoRows) {
		return nil, fmt.Errorf("error checking existing weddings: %w", err)
	}
	if len(existingWeddings) > 0 {
		return nil, errors.New("you can only have one wedding per account. Please delete your existing wedding to create a new one")
	}

	if len(req.CoupleNames) == 0 {
		return nil, errors.New("couple_names is required")
	}
	if req.Address == "" {
		return nil, errors.New("address is required")
	}
	if req.WhatsAppNumber == "" {
		return nil, errors.New("whatsapp_number is required")
	}
	if len(req.Ages) == 0 {
		return nil, errors.New("ages is required")
	}
	if len(req.CoupleNames) != len(req.Ages) {
		return nil, errors.New("couple_names and ages must have the same number of entries")
	}
	weddingDate, err := time.Parse("2006-01-02", req.WeddingDate)
	if err != nil {
		return nil, errors.New("wedding_date must be in YYYY-MM-DD format")
	}

	slug := functions.GenerateSlug()
	qrURL := fmt.Sprintf("%s/w/%s", s.cfg.FrontendURL, slug)
	qrCodeURL, _ := functions.GenerateQRDataURL(qrURL)

	now := time.Now()
	w := &dto.Wedding{
		ID:             uuid.NewString(),
		OwnerID:        ownerID,
		CoupleNames:    dto.StringArray(req.CoupleNames),
		WeddingDate:    weddingDate,
		Venue:          req.Venue,
		Address:        req.Address,
		WhatsAppNumber: req.WhatsAppNumber,
		Ages:           dto.IntArray(req.Ages),
		WelcomeMessage: req.WelcomeMessage,
		WeddingTime:    req.WeddingTime,
		Template:       req.Template,
		Lighting:       req.Lighting,
		StoryStyle:     req.StoryStyle,
		CeremonyStyle:  req.CeremonyStyle,
		VenueType:      req.VenueType,
		WeddingMood:    req.WeddingMood,
		WeddingTheme:   req.WeddingTheme,
		QRSlug:         slug,
		QRCodeURL:      qrCodeURL,
		Tier:           dto.TierElopement,
		Privacy:        dto.PrivacyPublic,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := dao.CreateWedding(ctx, s.db, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *WeddingService) List(ctx context.Context, ownerID string) ([]*dto.Wedding, error) {
	return dao.FindWeddingsByOwner(ctx, s.db, ownerID)
}

func (s *WeddingService) Get(ctx context.Context, weddingID, ownerID string) (*dto.Wedding, error) {
	w, err := dao.FindWeddingByID(ctx, s.db, weddingID)
	if err != nil {
		return nil, err
	}
	if w.OwnerID != ownerID {
		return nil, apperrors.ErrForbidden
	}
	return w, nil
}

func (s *WeddingService) GetBySlug(ctx context.Context, slug string) (*dto.Wedding, error) {
	return dao.FindWeddingBySlug(ctx, s.db, slug)
}

func (s *WeddingService) GetPublicByID(ctx context.Context, weddingID string) (*dto.Wedding, error) {
	w, err := dao.FindWeddingByID(ctx, s.db, weddingID)
	if err != nil {
		return nil, err
	}
	if w.Privacy != dto.PrivacyPublic {
		return nil, apperrors.ErrForbidden
	}
	w.PasswordHash = nil
	return w, nil
}

func (s *WeddingService) GetPublicByIdentifier(ctx context.Context, identifier string) (*dto.Wedding, error) {
	w, err := dao.FindWeddingByID(ctx, s.db, identifier)
	if err != nil {
		if !errors.Is(err, dao.ErrNoRows) {
			return nil, err
		}
		w, err = dao.FindWeddingBySlug(ctx, s.db, identifier)
		if err != nil {
			return nil, err
		}
	}
	if w.Privacy != dto.PrivacyPublic {
		return nil, apperrors.ErrForbidden
	}
	w.PasswordHash = nil
	return w, nil
}

func (s *WeddingService) VerifyGuestAccessByIdentifier(ctx context.Context, identifier, password string) (*dto.Wedding, error) {
	w, err := dao.FindWeddingByID(ctx, s.db, identifier)
	if err != nil {
		if !errors.Is(err, dao.ErrNoRows) {
			return nil, err
		}
		w, err = dao.FindWeddingBySlug(ctx, s.db, identifier)
		if err != nil {
			return nil, err
		}
	}
	if w.Privacy == dto.PrivacyPrivate {
		return nil, apperrors.ErrForbidden
	}
	if w.Privacy == dto.PrivacyPasswordProtected {
		if w.PasswordHash == nil || !functions.CheckPassword(*w.PasswordHash, password) {
			return nil, errors.New("incorrect album password")
		}
	}
	return w, nil
}

func (s *WeddingService) Update(ctx context.Context, weddingID, ownerID string, req dto.UpdateWeddingRequest) (*dto.Wedding, error) {
	w, err := s.Get(ctx, weddingID, ownerID)
	if err != nil {
		return nil, err
	}
	if req.CoupleNames != nil {
		w.CoupleNames = *req.CoupleNames
	}
	if req.Venue != nil {
		w.Venue = *req.Venue
	}
	if req.Address != nil {
		w.Address = *req.Address
	}
	if req.WhatsAppNumber != nil {
		w.WhatsAppNumber = *req.WhatsAppNumber
	}
	if req.Ages != nil {
		w.Ages = dto.IntArray(*req.Ages)
	}
	if req.WelcomeMessage != nil {
		w.WelcomeMessage = *req.WelcomeMessage
	}
	if req.Template != nil {
		w.Template = *req.Template
	}
	if req.Lighting != nil {
		w.Lighting = *req.Lighting
	}
	if req.StoryStyle != nil {
		w.StoryStyle = *req.StoryStyle
	}
	if req.CeremonyStyle != nil {
		w.CeremonyStyle = *req.CeremonyStyle
	}
	if req.VenueType != nil {
		w.VenueType = *req.VenueType
	}
	if req.WeddingMood != nil {
		w.WeddingMood = *req.WeddingMood
	}
	if req.WeddingTheme != nil {
		w.WeddingTheme = *req.WeddingTheme
	}
	if req.WeddingTime != nil {
		if *req.WeddingTime != "" {
			if _, err := time.Parse("15:04", *req.WeddingTime); err != nil {
				return nil, errors.New("wedding_time must be in HH:MM format")
			}
		}
		w.WeddingTime = *req.WeddingTime
	}
	if req.WeddingDate != nil {
		d, err := time.Parse("2006-01-02", *req.WeddingDate)
		if err != nil {
			return nil, errors.New("wedding_date must be YYYY-MM-DD")
		}
		w.WeddingDate = d
	}
	if err := dao.UpdateWedding(ctx, s.db, w); err != nil {
		return nil, err
	}
	return w, nil
}

// Delete permanently removes a wedding and everything scoped to it.
//
// This used to call DeactivateWedding, which only set is_active=false. Since
// FindWeddingsByOwner doesn't filter on that flag, the "deleted" wedding stayed
// in the owner's list forever — and because Create rejects a second wedding, the
// owner was left unable to delete it or replace it.
func (s *WeddingService) Delete(ctx context.Context, weddingID, ownerID string) error {
	if _, err := s.Get(ctx, weddingID, ownerID); err != nil {
		return err
	}

	// Clear stored objects first: the upload rows are the only record of which
	// S3 keys belong to this wedding.
	if s.uploads != nil {
		s.uploads.DeleteObjectsForWedding(ctx, weddingID)
	}

	return dao.DeleteWedding(ctx, s.db, weddingID)
}

func (s *WeddingService) SetPrivacy(ctx context.Context, weddingID, ownerID string, req dto.PrivacyRequest) error {
	if _, err := s.Get(ctx, weddingID, ownerID); err != nil {
		return err
	}
	var passwordHash *string
	if req.Privacy == dto.PrivacyPasswordProtected {
		if req.Password == nil || *req.Password == "" {
			return errors.New("password required for password_protected privacy")
		}
		h, err := functions.HashPassword(*req.Password)
		if err != nil {
			return err
		}
		passwordHash = &h
	}
	return dao.UpdateWeddingPrivacy(ctx, s.db, weddingID, req.Privacy, passwordHash)
}

func (s *WeddingService) VerifyGuestAccess(ctx context.Context, slug, password string) (*dto.Wedding, error) {
	w, err := dao.FindWeddingBySlug(ctx, s.db, slug)
	if err != nil {
		return nil, err
	}
	if w.Privacy == dto.PrivacyPrivate {
		return nil, apperrors.ErrForbidden
	}
	if w.Privacy == dto.PrivacyPasswordProtected {
		if w.PasswordHash == nil || !functions.CheckPassword(*w.PasswordHash, password) {
			return nil, errors.New("incorrect album password")
		}
	}
	return w, nil
}

func (s *WeddingService) ActivateTier(ctx context.Context, weddingID string, tier dto.Tier, expiresAt *time.Time) error {
	return dao.ActivateWeddingTier(ctx, s.db, weddingID, tier, expiresAt)
}
