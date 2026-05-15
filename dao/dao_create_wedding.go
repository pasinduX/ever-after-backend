package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateWedding(ctx context.Context, db *mongo.Database, w *dto.Wedding) error {
	_, err := db.Collection("weddings").InsertOne(ctx, w)
	if err != nil {
		return fmt.Errorf("dao CreateWedding: %w", err)
	}
	return nil
}
