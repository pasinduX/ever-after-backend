package dao

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

func MarkOrderPaid(ctx context.Context, db *mongo.Database, stripeSessionID, paymentIntentID string) error {
	update := bson.M{"status": "paid", "paid_at": time.Now()}
	if paymentIntentID != "" {
		update["stripe_payment_intent_id"] = paymentIntentID
	}
	_, err := db.Collection("orders").UpdateOne(ctx, bson.M{"stripe_session_id": stripeSessionID}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("dao MarkOrderPaid: %w", err)
	}
	return nil
}
