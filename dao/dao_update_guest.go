package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func UpdateGuest(ctx context.Context, db *mongo.Database, g *dto.Guest) error {
	_, err := db.Collection("guests").ReplaceOne(ctx, bson.M{"_id": g.ID}, g)
	if err != nil {
		return fmt.Errorf("dao UpdateGuest: %w", err)
	}
	return nil
}
