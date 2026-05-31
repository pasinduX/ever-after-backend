package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateGuest(ctx context.Context, db *mongo.Database, g *dto.Guest) error {
	_, err := db.Collection("guests").InsertOne(ctx, g)
	if err != nil {
		return fmt.Errorf("dao CreateGuest: %w", err)
	}
	return nil
}
