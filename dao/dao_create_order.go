package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateOrder(ctx context.Context, db *mongo.Database, o *dto.Order) error {
	_, err := db.Collection("orders").InsertOne(ctx, o)
	if err != nil {
		return fmt.Errorf("dao CreateOrder: %w", err)
	}
	return nil
}
