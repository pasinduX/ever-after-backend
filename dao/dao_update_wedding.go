package dao

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"time"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
)

func UpdateWedding(ctx context.Context, db *mongo.Database, w *dto.Wedding) error {
	w.UpdatedAt = time.Now()
	_, err := db.Collection("weddings").ReplaceOne(ctx, bson.M{"_id": w.ID}, w)
	if err != nil {
		return fmt.Errorf("dao UpdateWedding: %w", err)
	}
	return nil
}

func UpdateWeddingPrivacy(ctx context.Context, db *mongo.Database, weddingID string, privacy dto.Privacy, passwordHash *string) error {
	update := bson.M{"privacy": privacy}
	if passwordHash != nil {
		update["password_hash"] = *passwordHash
	} else {
		update["password_hash"] = nil
	}
	_, err := db.Collection("weddings").UpdateOne(ctx, bson.M{"_id": weddingID}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("dao UpdateWeddingPrivacy: %w", err)
	}
	return nil
}

func ActivateWeddingTier(ctx context.Context, db *mongo.Database, weddingID string, tier dto.Tier, expiresAt *time.Time) error {
	update := bson.M{"tier": tier, "expires_at": expiresAt}
	_, err := db.Collection("weddings").UpdateOne(ctx, bson.M{"_id": weddingID}, bson.M{"$set": update})
	if err != nil {
		return fmt.Errorf("dao ActivateWeddingTier: %w", err)
	}
	return nil
}
