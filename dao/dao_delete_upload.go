package dao

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/bson"
)

func DeleteUpload(ctx context.Context, db *mongo.Database, uploadID string) error {
	_, err := db.Collection("uploads").DeleteOne(ctx, bson.M{"_id": uploadID})
	if err != nil {
		return fmt.Errorf("dao DeleteUpload: %w", err)
	}
	return nil
}
