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

type GuestService struct {
	db *mongo.Database
}

func NewGuestService(db *mongo.Database) *GuestService {
	return &GuestService{db: db}
}

func (s *GuestService) ensureWeddingOwner(ctx context.Context, ownerID, weddingID string) error {
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

func (s *GuestService) Create(ctx context.Context, userID, weddingID string, req dto.CreateGuestRequest) (*dto.Guest, error) {
	if err := s.ensureWeddingOwner(ctx, userID, weddingID); err != nil {
		return nil, err
	}
	if req.CaptainName == "" {
		return nil, errors.New("captain_name is required")
	}
	if req.Side != dto.GuestSideBride && req.Side != dto.GuestSideGroom && req.Side != dto.GuestSideBoth {
		return nil, errors.New("side must be one of 'bride', 'groom', 'both'")
	}
	membersInvited := 1
	if req.MembersInvited != nil {
		membersInvited = *req.MembersInvited
	}
	if membersInvited < 1 {
		return nil, errors.New("members_invited must be at least 1")
	}
	membersComing := 0
	if req.MembersComing != nil {
		membersComing = *req.MembersComing
	}
	if membersComing < 0 {
		return nil, errors.New("members_coming cannot be negative")
	}
	status := dto.GuestStatusPending
	if req.Status != "" {
		if req.Status != dto.GuestStatusPending && req.Status != dto.GuestStatusConfirmed && req.Status != dto.GuestStatusDeclined {
			return nil, errors.New("status must be one of 'pending', 'confirmed', 'declined'")
		}
		status = req.Status
	}
	guest := &dto.Guest{
		ID:             uuid.NewString(),
		UserID:         userID,
		WeddingID:      weddingID,
		CaptainName:    req.CaptainName,
		Phone:          req.Phone,
		Side:           req.Side,
		MembersInvited: membersInvited,
		MembersComing:  membersComing,
		Status:         status,
		Notes:          "",
		CreatedAt:      time.Now(),
	}
	if req.Notes != nil {
		guest.Notes = *req.Notes
	}
	if err := dao.CreateGuest(ctx, s.db, guest); err != nil {
		return nil, err
	}
	return guest, nil
}

func (s *GuestService) List(ctx context.Context, ownerID, weddingID string) ([]*dto.Guest, error) {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return nil, err
	}
	return dao.FindGuestsByWedding(ctx, s.db, weddingID)
}

func (s *GuestService) Get(ctx context.Context, ownerID, weddingID, guestID string) (*dto.Guest, error) {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return nil, err
	}
	g, err := dao.FindGuestByID(ctx, s.db, guestID)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return nil, apperrors.ErrGuestNotFound
		}
		return nil, err
	}
	if g.WeddingID != weddingID {
		return nil, apperrors.ErrForbidden
	}
	return g, nil
}

func (s *GuestService) Update(ctx context.Context, ownerID, weddingID, guestID string, req dto.UpdateGuestRequest) (*dto.Guest, error) {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return nil, err
	}
	g, err := dao.FindGuestByID(ctx, s.db, guestID)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return nil, apperrors.ErrGuestNotFound
		}
		return nil, err
	}
	if g.WeddingID != weddingID {
		return nil, apperrors.ErrForbidden
	}
	if req.CaptainName != nil {
		g.CaptainName = *req.CaptainName
	}
	if req.Phone != nil {
		g.Phone = *req.Phone
	}
	if req.Side != nil {
		if *req.Side != dto.GuestSideBride && *req.Side != dto.GuestSideGroom && *req.Side != dto.GuestSideBoth {
			return nil, errors.New("side must be one of 'bride', 'groom', 'both'")
		}
		g.Side = *req.Side
	}
	if req.MembersInvited != nil {
		if *req.MembersInvited < 1 {
			return nil, errors.New("members_invited must be at least 1")
		}
		g.MembersInvited = *req.MembersInvited
	}
	if req.MembersComing != nil {
		if *req.MembersComing < 0 {
			return nil, errors.New("members_coming cannot be negative")
		}
		g.MembersComing = *req.MembersComing
	}
	if req.Status != nil {
		if *req.Status != dto.GuestStatusPending && *req.Status != dto.GuestStatusConfirmed && *req.Status != dto.GuestStatusDeclined {
			return nil, errors.New("status must be one of 'pending', 'confirmed', 'declined'")
		}
		g.Status = *req.Status
	}
	if req.Notes != nil {
		g.Notes = *req.Notes
	}
	if err := dao.UpdateGuest(ctx, s.db, g); err != nil {
		return nil, err
	}
	return g, nil
}

func (s *GuestService) Delete(ctx context.Context, ownerID, weddingID, guestID string) error {
	if err := s.ensureWeddingOwner(ctx, ownerID, weddingID); err != nil {
		return err
	}
	g, err := dao.FindGuestByID(ctx, s.db, guestID)
	if err != nil {
		if errors.Is(err, dao.ErrNoRows) {
			return apperrors.ErrGuestNotFound
		}
		return err
	}
	if g.WeddingID != weddingID {
		return apperrors.ErrForbidden
	}
	return dao.DeleteGuest(ctx, s.db, guestID)
}
