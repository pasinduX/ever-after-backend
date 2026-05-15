package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func FindRefreshTokenByHash(ctx context.Context, db *mongo.Database, tokenHash string) (*dto.RefreshToken, error) {
	var rt dto.RefreshToken
	err := db.Collection("refresh_tokens").FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(&rt)
	if err != nil {
		return nil, fmt.Errorf("dao FindRefreshTokenByHash: %w", err)
	}
	return &rt, nil
}
