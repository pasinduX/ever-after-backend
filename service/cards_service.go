package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	apperrors "github.com/storyvows/backend/errors"
	"go.mongodb.org/mongo-driver/mongo"
)

type CardsService struct {
	db       *mongo.Database
	analysis *AnalysisService
}

func NewCardsService(db *mongo.Database, analysis *AnalysisService) *CardsService {
	return &CardsService{db: db, analysis: analysis}
}

func (s *CardsService) ensureWeddingOwner(ctx context.Context, ownerID, weddingID string) error {
	w, err := dao.FindWeddingByID(ctx, s.db, weddingID)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return apperrors.ErrWeddingNotFound
		}
		return err
	}
	if w.OwnerID != ownerID {
		return apperrors.ErrForbidden
	}
	return nil
}

func (s *CardsService) getWeddingForOwner(ctx context.Context, ownerID, weddingID string) (*dto.Wedding, error) {
	w, err := dao.FindWeddingByID(ctx, s.db, weddingID)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return nil, apperrors.ErrWeddingNotFound
		}
		return nil, err
	}
	if w.OwnerID != ownerID {
		return nil, apperrors.ErrForbidden
	}
	return w, nil
}

func (s *CardsService) GenerateInvite(ctx context.Context, ownerID, weddingID string) (*dto.InviteConfig, error) {
	if s.analysis == nil {
		return nil, errors.New("analysis service is not configured")
	}
	wedding, err := s.getWeddingForOwner(ctx, ownerID, weddingID)
	if err != nil {
		return nil, err
	}
	req, err := s.analysis.GenerateInviteConfig(ctx, wedding)
	if err != nil {
		return nil, err
	}
	return s.CreateInvite(ctx, ownerID, weddingID, *req)
}

func (s *CardsService) GenerateThankYou(ctx context.Context, ownerID, weddingID string) (*dto.ThankYouConfig, error) {
	if s.analysis == nil {
		return nil, errors.New("analysis service is not configured")
	}
	wedding, err := s.getWeddingForOwner(ctx, ownerID, weddingID)
	if err != nil {
		return nil, err
	}
	req, err := s.analysis.GenerateThankYouConfig(ctx, wedding)
	if err != nil {
		return nil, err
	}
	return s.CreateThankYou(ctx, ownerID, weddingID, *req)
}

func (s *CardsService) CreateInvite(ctx context.Context, ownerID, weddingID string, req dto.CreateInviteRequest) (*dto.InviteConfig, error) {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return nil, err
	}
	now := time.Now()
	cfg := &dto.InviteConfig{
		ID:        uuid.NewString(),
		WeddingID: weddingID,
		Couple:    req.Couple,
		Hashtag:   req.Hashtag,
		Intro:     req.Intro,
		Story:     req.Story,
		Details:   req.Details,
		Countdown: req.Countdown,
		QR:        req.QR,
		Outro:     req.Outro,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := dao.CreateInviteConfig(ctx, s.db, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *CardsService) GetInvite(ctx context.Context, ownerID, weddingID string) (*dto.InviteConfig, error) {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return nil, err
	}
	cfg, err := dao.FindInviteConfigByWedding(ctx, s.db, weddingID)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return nil, apperrors.ErrInviteNotFound
		}
		return nil, err
	}
	return cfg, nil
}

func (s *CardsService) UpdateInvite(ctx context.Context, ownerID, weddingID string, req dto.UpdateInviteRequest) (*dto.InviteConfig, error) {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return nil, err
	}
	cfg, err := dao.FindInviteConfigByWedding(ctx, s.db, weddingID)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return nil, apperrors.ErrInviteNotFound
		}
		return nil, err
	}
	if req.Couple != nil {
		cfg.Couple = *req.Couple
	}
	if req.Hashtag != nil {
		cfg.Hashtag = *req.Hashtag
	}
	if req.Intro != nil {
		cfg.Intro = *req.Intro
	}
	if req.Story != nil {
		cfg.Story = *req.Story
	}
	if req.Details != nil {
		cfg.Details = *req.Details
	}
	if req.Countdown != nil {
		cfg.Countdown = *req.Countdown
	}
	if req.QR != nil {
		cfg.QR = *req.QR
	}
	if req.Outro != nil {
		cfg.Outro = *req.Outro
	}
	cfg.UpdatedAt = time.Now()
	if err := dao.UpdateInviteConfig(ctx, s.db, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *CardsService) DeleteInvite(ctx context.Context, ownerID, weddingID string) error {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return err
	}
	return dao.DeleteInviteConfig(ctx, s.db, weddingID)
}

func (s *CardsService) CreateThankYou(ctx context.Context, ownerID, weddingID string, req dto.CreateThankYouRequest) (*dto.ThankYouConfig, error) {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return nil, err
	}
	now := time.Now()
	cfg := &dto.ThankYouConfig{
		ID:        uuid.NewString(),
		WeddingID: weddingID,
		Template:  req.Template,
		Couple:    req.Couple,
		Date:      req.Date,
		Venue:     req.Venue,
		Hashtag:   req.Hashtag,
		HeroImage: req.HeroImage,
		Portrait:  req.Portrait,
		Intro:     req.Intro,
		Message:   req.Message,
		Signature: req.Signature,
		Gallery:   req.Gallery,
		Closing:   req.Closing,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := dao.CreateThankYouConfig(ctx, s.db, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *CardsService) GetThankYou(ctx context.Context, ownerID, weddingID string) (*dto.ThankYouConfig, error) {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return nil, err
	}
	cfg, err := dao.FindThankYouConfigByWedding(ctx, s.db, weddingID)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return nil, apperrors.ErrThankYouNotFound
		}
		return nil, err
	}
	return cfg, nil
}

func (s *CardsService) UpdateThankYou(ctx context.Context, ownerID, weddingID string, req dto.UpdateThankYouRequest) (*dto.ThankYouConfig, error) {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return nil, err
	}
	cfg, err := dao.FindThankYouConfigByWedding(ctx, s.db, weddingID)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return nil, apperrors.ErrThankYouNotFound
		}
		return nil, err
	}
	if req.Template != nil {
		cfg.Template = *req.Template
	}
	if req.Couple != nil {
		cfg.Couple = *req.Couple
	}
	if req.Date != nil {
		cfg.Date = *req.Date
	}
	if req.Venue != nil {
		cfg.Venue = *req.Venue
	}
	if req.Hashtag != nil {
		cfg.Hashtag = *req.Hashtag
	}
	if req.HeroImage != nil {
		cfg.HeroImage = *req.HeroImage
	}
	if req.Portrait != nil {
		cfg.Portrait = *req.Portrait
	}
	if req.Intro != nil {
		cfg.Intro = req.Intro
	}
	if req.Message != nil {
		cfg.Message = *req.Message
	}
	if req.Signature != nil {
		cfg.Signature = *req.Signature
	}
	if req.Gallery != nil {
		cfg.Gallery = req.Gallery
	}
	if req.Closing != nil {
		cfg.Closing = *req.Closing
	}
	cfg.UpdatedAt = time.Now()
	if err := dao.UpdateThankYouConfig(ctx, s.db, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (s *CardsService) DeleteThankYou(ctx context.Context, ownerID, weddingID string) error {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return err
	}
	return dao.DeleteThankYouConfig(ctx, s.db, weddingID)
}
