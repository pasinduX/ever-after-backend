package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateUpload(ctx context.Context, db *mongo.Database, u *dto.Upload) error {
	_, err := db.Collection("uploads").InsertOne(ctx, u)
	if err != nil {
		return fmt.Errorf("dao CreateUpload: %w", err)
	}
	return nil
}
