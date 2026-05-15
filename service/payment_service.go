package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	apperrors "github.com/storyvows/backend/errors"
	"github.com/storyvows/backend/integrations"
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"go.mongodb.org/mongo-driver/mongo"
)

type PaymentService struct {
	db  *mongo.Database
	cfg *integrations.Secrets
}

func NewPaymentService(db *mongo.Database, cfg *integrations.Secrets) *PaymentService {
	stripe.Key = cfg.StripeSecretKey
	return &PaymentService{db: db, cfg: cfg}
}

func (s *PaymentService) CreateCheckout(ctx context.Context, userID string, req dto.CheckoutRequest) (*dto.CheckoutResponse, error) {
	amount, name := s.tierDetails(req.Tier)
	if amount == 0 {
		return nil, apperrors.ErrInvalidTier
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String("usd"),
					UnitAmount: stripe.Int64(amount),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(fmt.Sprintf("Story Vows — %s", name)),
					},
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(fmt.Sprintf("%s/dashboard?payment=success", s.cfg.FrontendURL)),
		CancelURL:  stripe.String(fmt.Sprintf("%s/pricing?payment=cancelled", s.cfg.FrontendURL)),
		Metadata: map[string]string{
			"user_id":    userID,
			"wedding_id": req.WeddingID,
			"tier":       string(req.Tier),
		},
	}

	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("create stripe session: %w", err)
	}

	order := &dto.Order{
		ID:              uuid.NewString(),
		WeddingID:       req.WeddingID,
		UserID:          userID,
		Tier:            req.Tier,
		AmountCents:     amount,
		Currency:        "usd",
		StripeSessionID: sess.ID,
		Status:          dto.OrderStatusPending,
		CreatedAt:       time.Now(),
	}
	if err := dao.CreateOrder(ctx, s.db, order); err != nil {
		return nil, fmt.Errorf("save order: %w", err)
	}

	return &dto.CheckoutResponse{
		CheckoutURL: sess.URL,
		SessionID:   sess.ID,
	}, nil
}

func (s *PaymentService) HandleWebhook(ctx context.Context, event stripe.Event) error {
	if event.Type != "checkout.session.completed" {
		return nil
	}

	sess := &stripe.CheckoutSession{}
	if err := json.Unmarshal(event.Data.Raw, sess); err != nil {
		return fmt.Errorf("parse session: %w", err)
	}

	weddingID := sess.Metadata["wedding_id"]
	tier := dto.Tier(sess.Metadata["tier"])
	paymentIntentID := ""
	if sess.PaymentIntent != nil {
		paymentIntentID = sess.PaymentIntent.ID
	}

	if err := dao.MarkOrderPaid(ctx, s.db, sess.ID, paymentIntentID); err != nil {
		return fmt.Errorf("update order: %w", err)
	}

	var expiresAt *time.Time
	if tier == dto.TierElopement {
		t := time.Now().AddDate(1, 0, 0)
		expiresAt = &t
	}

	if err := dao.ActivateWeddingTier(ctx, s.db, weddingID, tier, expiresAt); err != nil {
		return fmt.Errorf("activate tier: %w", err)
	}
	return nil
}

func (s *PaymentService) tierDetails(tier dto.Tier) (int64, string) {
	switch tier {
	case dto.TierElopement:
		return s.cfg.StripeElopementPrice, "Elopement"
	case dto.TierHeritage:
		return s.cfg.StripeHeritagePrice, "Heritage"
	case dto.TierLegacy:
		return s.cfg.StripeLegacyPrice, "Legacy"
	}
	return 0, ""
}
