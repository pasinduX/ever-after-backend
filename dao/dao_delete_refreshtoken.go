package dao

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"

	"go.mongodb.org/mongo-driver/bson"
)

func DeleteRefreshTokenByID(ctx context.Context, db *mongo.Database, id string) error {
	_, err := db.Collection("refresh_tokens").DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("dao DeleteRefreshTokenByID: %w", err)
	}
	return nil
}

func DeleteRefreshTokenByHash(ctx context.Context, db *mongo.Database, tokenHash string) error {
	_, err := db.Collection("refresh_tokens").DeleteOne(ctx, bson.M{"token_hash": tokenHash})
	if err != nil {
		return fmt.Errorf("dao DeleteRefreshTokenByHash: %w", err)
	}
	return nil
}
