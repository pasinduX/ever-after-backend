package dao

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
)

func FindWeddingByID(ctx context.Context, db *mongo.Database, id string) (*dto.Wedding, error) {
	var w dto.Wedding
	err := db.Collection("weddings").FindOne(ctx, bson.M{"_id": id}).Decode(&w)
	if err != nil {
		return nil, fmt.Errorf("dao FindWeddingByID: %w", err)
	}
	return &w, nil
}

func FindWeddingBySlug(ctx context.Context, db *mongo.Database, slug string) (*dto.Wedding, error) {
	var w dto.Wedding
	err := db.Collection("weddings").FindOne(ctx, bson.M{"qr_slug": slug}).Decode(&w)
	if err != nil {
		return nil, fmt.Errorf("dao FindWeddingBySlug: %w", err)
	}
	return &w, nil
}
