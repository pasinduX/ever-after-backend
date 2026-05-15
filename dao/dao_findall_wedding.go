package dao

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/bson"
)

func FindWeddingsByOwner(ctx context.Context, db *mongo.Database, ownerID string) ([]*dto.Wedding, error) {
	cursor, err := db.Collection("weddings").Find(ctx, bson.M{"owner_id": ownerID})
	if err != nil {
		return nil, fmt.Errorf("dao FindWeddingsByOwner: %w", err)
	}
	defer cursor.Close(ctx)

	var weddings []*dto.Wedding
	if err := cursor.All(ctx, &weddings); err != nil {
		return nil, fmt.Errorf("dao FindWeddingsByOwner decode: %w", err)
	}
	return weddings, nil
}
