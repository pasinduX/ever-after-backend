package dao

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/bson"
)

func SetUploadApproval(ctx context.Context, db *mongo.Database, uploadID string, approved bool) error {
	_, err := db.Collection("uploads").UpdateOne(ctx, bson.M{"_id": uploadID}, bson.M{"$set": bson.M{"is_approved": approved}})
	if err != nil {
		return fmt.Errorf("dao SetUploadApproval: %w", err)
	}
	return nil
}
