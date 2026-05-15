package dto

import "time"

type OrderStatus string

const (
	OrderStatusPending  OrderStatus = "pending"
	OrderStatusPaid     OrderStatus = "paid"
	OrderStatusRefunded OrderStatus = "refunded"
)

type Order struct {
	ID                    string      `json:"id" bson:"_id,omitempty"`
	WeddingID             string      `json:"wedding_id" bson:"wedding_id"`
	UserID                string      `json:"user_id" bson:"user_id"`
	Tier                  Tier        `json:"tier" bson:"tier"`
	AmountCents           int64       `json:"amount_cents" bson:"amount_cents"`
	Currency              string      `json:"currency" bson:"currency"`
	Status                OrderStatus `json:"status" bson:"status"`
	StripeSessionID       string      `json:"stripe_session_id" bson:"stripe_session_id"`
	StripePaymentIntentID *string     `json:"stripe_payment_intent_id,omitempty" bson:"stripe_payment_intent_id,omitempty"`
	PaidAt                *time.Time  `json:"paid_at,omitempty" bson:"paid_at,omitempty"`
	ExpiresAt             *time.Time  `json:"expires_at,omitempty" bson:"expires_at,omitempty"`
	CreatedAt             time.Time   `json:"created_at" bson:"created_at"`
}

type CheckoutRequest struct {
	WeddingID string `json:"wedding_id"`
	Tier      Tier   `json:"tier"`
}

type CheckoutResponse struct {
	CheckoutURL string `json:"checkout_url"`
	SessionID   string `json:"session_id"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

type SuccessResponse struct {
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}
