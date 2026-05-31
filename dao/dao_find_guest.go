package dao

import (
	"context"
	"fmt"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func FindGuestByID(ctx context.Context, db *mongo.Database, id string) (*dto.Guest, error) {
	var g dto.Guest
	err := db.Collection("guests").FindOne(ctx, bson.M{"_id": id}).Decode(&g)
	if err != nil {
		return nil, fmt.Errorf("dao FindGuestByID: %w", err)
	}
	return &g, nil
}

func FindGuestsByWedding(ctx context.Context, db *mongo.Database, weddingID string) ([]*dto.Guest, error) {
	cursor, err := db.Collection("guests").Find(ctx, bson.M{"wedding_id": weddingID})
	if err != nil {
		return nil, fmt.Errorf("dao FindGuestsByWedding: %w", err)
	}
	defer cursor.Close(ctx)

	var guests []*dto.Guest
	if err := cursor.All(ctx, &guests); err != nil {
		return nil, fmt.Errorf("dao FindGuestsByWedding decode: %w", err)
	}
	return guests, nil
}
