package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateRefreshToken(ctx context.Context, db *mongo.Database, rt *dto.RefreshToken) error {
	_, err := db.Collection("refresh_tokens").InsertOne(ctx, rt)
	if err != nil {
		return fmt.Errorf("dao CreateRefreshToken: %w", err)
	}
	return nil
}
