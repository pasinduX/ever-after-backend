package dao

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/bson"
)

func DeactivateWedding(ctx context.Context, db *mongo.Database, weddingID string) error {
	_, err := db.Collection("weddings").UpdateOne(ctx, bson.M{"_id": weddingID}, bson.M{"$set": bson.M{"is_active": false}})
	if err != nil {
		return fmt.Errorf("dao DeactivateWedding: %w", err)
	}
	return nil
}
