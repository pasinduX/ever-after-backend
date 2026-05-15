package dao

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func DeleteUserByEmail(ctx context.Context, db *mongo.Database, email string) error {
	_, err := db.Collection("users").DeleteOne(ctx, bson.M{"email": email})
	if err != nil {
		return fmt.Errorf("dao DeleteUserByEmail: %w", err)
	}
	return nil
}
