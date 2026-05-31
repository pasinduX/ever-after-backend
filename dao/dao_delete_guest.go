package dao

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func DeleteGuest(ctx context.Context, db *mongo.Database, id string) error {
	_, err := db.Collection("guests").DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("dao DeleteGuest: %w", err)
	}
	return nil
}
